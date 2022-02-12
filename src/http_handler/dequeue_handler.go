package http_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
)

type dequeueFunc func(ctx context.Context, queueID string) (gen.DequeueRow, error)

func dequeueHandler(dequeue dequeueFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		queueID := chi.URLParam(r, "queueID")

		l := logger.With().
			Str("queue_id", queueID).
			Logger()
		logger = &l
		ctx = logger.WithContext(ctx)

		msg, err := dequeue(ctx, queueID)
		if err != nil {

			switch {
			case errors.Is(err, pgx.ErrNoRows):
				logger.Err(err).Send()
				rw.WriteHeader(http.StatusNotFound)

			default:
				logger.Err(err).Send()
				rw.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		rw.Header().Add("content-type", "application/json")
		if err := json.NewEncoder(rw).Encode(&msg); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
