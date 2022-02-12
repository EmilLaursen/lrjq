package http_handler

import (
	"net/http"
	"time"

	"github.com/EmilLaursen/lrjq/openapi"
	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func QueueRouter(r *chi.Mux, logger *zerolog.Logger, pool *pgxpool.Pool) *chi.Mux {
	r.Use(NewHandlerLoggingChain(logger))
	r.Post("/queue/enqueue/{queueID}", enqueueHandler(gen.NewQuerier(pool).Enqueue))
	r.Get("/queue/dequeue/{queueID}", dequeueHandler(gen.NewQuerier(pool).Dequeue))
	r.Post("/queue/heartbeat/{ID}/{workID}", heartbeatHandler(gen.NewQuerier(pool).SendHeartBeat))
	r.Put("/queue/ack/{ID}/{workID}", reportDoneHandler(gen.NewQuerier(pool).ReportDone))
	r.Mount("/", openapi.OpenAPIHandler())
	return r
}

func NewHandlerLoggingChain(logger *zerolog.Logger) func(handler http.Handler) http.Handler {
	middlewareChain := alice.New()
	handlerLogger := hlog.NewHandler(*logger)
	middlewareChain = middlewareChain.Append(handlerLogger)

	if addAccessLogger := logger.Debug().Enabled(); addAccessLogger {
		middlewareChain = middlewareChain.Append(
			hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
				hlog.FromRequest(r).Debug().
					Str("method", r.Method).
					Stringer("url", r.URL).
					Int("status", status).
					Int("size", size).
					Dur("duration", duration).
					Send()
			}))
	}

	middlewareChain = middlewareChain.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	return func(handler http.Handler) http.Handler {
		return middlewareChain.ThenFunc(handler.ServeHTTP)
	}
}
