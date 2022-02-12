package openapi

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/go-chi/chi/v5"
	"github.com/rotisserie/eris"
)

var ErrRouteContextNotFound = eris.New("route ctx not found")

func findr(doc *openapi3.T, r *http.Request) (*routers.Route, map[string]string, error) {
	routeCtx := chi.RouteContext(r.Context())
	if routeCtx == nil {
		return nil, nil, ErrRouteContextNotFound
	}
	pathParams := map[string]string{}
	for k, key := range routeCtx.URLParams.Keys {
		pathParams[key] = routeCtx.URLParams.Values[k]
	}
	pi := doc.Paths.Find(routeCtx.RoutePattern())
	route := &routers.Route{
		Spec:      doc,
		Path:      routeCtx.RoutePattern(),
		PathItem:  pi,
		Method:    routeCtx.RouteMethod,
		Operation: pi.GetOperation(routeCtx.RouteMethod),
	}
	return route, pathParams, nil
}

func Validator(doc *openapi3.T) func(r *http.Request) error {
	return func(r *http.Request) error {
		route, pathParams, err := findr(doc, r)
		if err != nil {
			return err
		}
		input := openapi3filter.RequestValidationInput{
			Request:     r,
			PathParams:  pathParams,
			QueryParams: r.URL.Query(),
			Route:       route,
			Options:     &openapi3filter.Options{MultiError: true},
		}
		return openapi3filter.ValidateRequest(r.Context(), &input)
	}
}

type Middleware func(next http.Handler) http.Handler

func OpenAPIValidation(doc *openapi3.T) Middleware {
	validator := Validator(doc)
	return func(next http.Handler) http.Handler {
		var hf http.HandlerFunc = func(rw http.ResponseWriter, r *http.Request) {
			if err := validator(r); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte(err.Error()))
				return
			}
			next.ServeHTTP(rw, r)
		}
		return hf
	}
}
