package http_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
	"github.com/rs/zerolog"
)

func parseParams(r *http.Request) gen.EnqueueParams {
	logger := zerolog.Ctx(r.Context())

	if r == nil {
		return gen.EnqueueParams{}
	}

	var priority int32
	priorityRaw := r.URL.Query().Get("priority")
	if priorityRaw != "" {
		i, err := strconv.ParseInt(priorityRaw, 10, 32)
		if err != nil {
			logger.Err(err).Send()
		} else {
			priority = int32(i)
		}
	}

	// TODO: handle headers - not yet in model
	var headers map[string]string = map[string]string{}
	for h, v := range r.Header {
		if strings.HasPrefix(h, "X-LRJQ-") {
			if len(v) > 0 {
				headers[h] = v[0]
			}
		}
	}

	var hds bytes.Buffer
	err := json.NewEncoder(&hds).Encode(&headers)
	if err != nil {
		logger.Err(err).Send()
	}

	pl, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Err(err).Send()

	}

	return gen.EnqueueParams{
		Payload: pgtype.JSONB{
			Bytes:  pl,
			Status: pgtype.Present,
		},
		Priority: priority,
		QueueID:  chi.URLParam(r, "queueID"),
	}
}

type enqueueFunc func(ctx context.Context, msg gen.EnqueueParams) (gen.EnqueueRow, error)

func enqueueHandler(enqueue enqueueFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		params := parseParams(r)

		l := logger.With().
			Str("queue_id", params.QueueID).
			Int32("priority", params.Priority).
			Logger()
		logger = &l
		ctx = logger.WithContext(ctx)

		msg, err := enqueue(ctx, params)
		if err != nil {
			logger.Err(err).Send()
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Header().Add("content-type", "application/json")
		if err := json.NewEncoder(rw).Encode(&msg); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
