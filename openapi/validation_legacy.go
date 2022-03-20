package openapi

import (
	"fmt"
	"log"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/rs/zerolog"
)

type Middleware func(http.Handler) http.Handler

func NewValidator(router routers.Router) *ValidatorLegacy {
	return &ValidatorLegacy{router: router}
}

type ValidatorLegacy struct {
	router routers.Router
}

func (v *ValidatorLegacy) ValidateRequest(r *http.Request) error {

	route, pathParams, err := v.router.FindRoute(r)
	if err != nil {
		zerolog.Ctx(r.Context()).
			Err(err).
			Msgf("openapi request validator could not find route")
		return err
	}

	pps := make(map[string]interface{})
	for k, v := range pathParams {
		pps[k] = v
	}

	l := zerolog.Ctx(r.Context()).With().
		Str("operation_id", route.Operation.OperationID).
		Fields(pps).Logger()
	logger := &l

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:     r,
		QueryParams: r.URL.Query(),
		PathParams:  pathParams,
		Route:       route,
		Options:     &openapi3filter.Options{MultiError: true},
	}

	if err := openapi3filter.ValidateRequest(r.Context(), requestValidationInput); err != nil {
		logger.Debug().AnErr("validation_error", err).Msgf("openapi request invalid")
		return err
	}
	return nil
}

func OpenAPIValidationLegacy(router routers.Router) Middleware {
	v := &ValidatorLegacy{router: router}
	return func(next http.Handler) http.Handler {
		var handlerFunc http.HandlerFunc = func(rw http.ResponseWriter, r *http.Request) {
			if err := v.ValidateRequest(r); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				msg := fmt.Sprintf(`{"error": %q}`, err.Error())
				rw.Write([]byte(msg))
				return
			}
			next.ServeHTTP(rw, r)
		}
		return handlerFunc
	}
}

func GetOpenAPIMiddleware(doc *openapi3.T) (Middleware, error) {
	router, err := legacyrouter.NewRouter(doc)
	if err != nil {
		log.Printf("load openapi new router err: %s", err)
		return nil, err
	}
	return OpenAPIValidationLegacy(router), nil
}
