package openapi

import (
	"context"
	"embed"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test_*.yaml
var testFS embed.FS

func legacyValidator(doc *openapi3.T) func(r *http.Request) error {
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return func(r *http.Request) error { return err }
	}
	return func(r *http.Request) error {
		route, pa, err := router.FindRoute(r)
		if err != nil {
			return err
		}
		return openapi3filter.ValidateRequest(r.Context(), &openapi3filter.RequestValidationInput{
			Request:    r,
			PathParams: pa,
			Route:      route,
			Options:    &openapi3filter.Options{MultiError: true},
		})
	}
}

func TestValidationVsLegacyRouter(t *testing.T) {
	ctx := context.Background()

	doc, err := FromFS(ctx, testFS, "test_spec_1.yaml")
	require.Nil(t, err)

	legacyValidator := legacyValidator(doc)
	validator := Validator(doc)

	mkReq := func(url string, body io.Reader) *http.Request {
		treq := httptest.NewRequest(http.MethodPost, url, body)
		treq.Header.Add("content-type", "application/json")
		return treq
	}
	tests := []struct {
		name string
		req  *http.Request
	}{
		{"green", mkReq(
			"https://lol-server.com/pkg/mypkg/lol?orderBy=trolo",
			strings.NewReader(`"abcdef"`))},

		{"query missing", mkReq(
			"https://lol-server.com/pkg/mypkg/lol",
			strings.NewReader(`"abcdef"`))},

		{"body missing", mkReq(
			"https://lol-server.com/pkg/mypkg/lol?orderBy=lala",
			http.NoBody)},
	}

	r := chi.NewRouter()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rr *http.Request
			r.Post("/pkg/{pkgName}/lol", func(rw http.ResponseWriter, r *http.Request) { rr = r })
			exp := legacyValidator(tc.req)
			r.ServeHTTP(httptest.NewRecorder(), tc.req)
			require.NotNil(t, rr)
			actual := validator(rr)
			assert.ErrorIs(t, exp, actual)
		})
	}
}

func TestValidationChi(t *testing.T) {
	ctx := context.Background()

	doc, err := FromFS(ctx, testFS, "test_spec_1.yaml")
	require.Nil(t, err)

	router, err := legacy.NewRouter(doc)
	require.Nil(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"https://lol-server.com/pkg/mypkg/lol?orderBy=trolo",
		http.NoBody)

	route, pathParams, err := router.FindRoute(req)
	require.Nil(t, err)

	var rr *http.Request
	r := chi.NewRouter()
	r.Post("/pkg/{pkgName}/lol", func(rw http.ResponseWriter, r *http.Request) { rr = r })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rr == nil {
		t.Fatalf("not found")
	}
	_, apathParams, aerr := findr(doc, rr)
	require.Nil(t, aerr)
	assert.NotEmpty(t, apathParams)
	assert.Equal(t, apathParams, pathParams)

	cctx := chi.RouteContext(rr.Context())
	assert.Equal(t, "POST", cctx.RouteMethod)
	assert.Equal(t, route.Path, cctx.RoutePattern())

	pathItem := doc.Paths[cctx.RoutePattern()]
	require.NotNil(t, pathItem)

	assert.Equal(t, route.Method, cctx.RouteMethod)

	op := pathItem.GetOperation(cctx.RouteMethod)

	assert.Equal(t, route.Operation, op)
}
