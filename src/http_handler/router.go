package http_handler

import (
	"context"
	"io"
	"log"
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

func DrainAndClose(r *http.Request) {
	if r == nil {
		return
	}
	defer r.Body.Close()
	n, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		zerolog.Ctx(r.Context()).Debug().Int64("discarded", n).Err(err).Msg("drain error")
	}
}

func QueueRouter(r *chi.Mux, logger *zerolog.Logger, pool *pgxpool.Pool) *chi.Mux {
	m := NewHandlerLoggingChain(logger)
	r.Use(m)
	ctx := context.Background()
	doc, err := openapi.FromFS(logger.WithContext(ctx), openapi.OpenAPIFS, "openapi.yaml")
	if err != nil {
		log.Panic(err)
	}
	oaval, err := openapi.GetOpenAPIMiddleware(doc)
	if err != nil {
		log.Panic(err)
	}

	r.Use(oaval)
	r.Post("/queue/enqueue/{queueID}", enqueueHandler(gen.NewQuerier(pool).Enqueue).ServeHTTP)
	r.Get("/queue/dequeue/{queueID}", dequeueHandler(gen.NewQuerier(pool).Dequeue).ServeHTTP)
	r.Post("/queue/heartbeat/{ID}/{workID}", heartbeatHandler(gen.NewQuerier(pool).SendHeartBeat).ServeHTTP)
	r.Put("/queue/ack/{ID}/{workID}", reportDoneHandler(gen.NewQuerier(pool).ReportDone).ServeHTTP)

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
