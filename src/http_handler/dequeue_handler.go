package http_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
)

type MessageMeta struct {
	ID            int32              `json:"id"`
	QueueID       string             `json:"queue-id"`
	WorkSignature pgtype.UUID        `json:"work-signature"`
	CreatedAt     pgtype.Timestamptz `json:"created-at"`
	StartedAt     pgtype.Timestamptz `json:"started-at"`
	Tries         int32              `json:"tries"`
	Priority      int32              `json:"priority"`
	Status        gen.JobStatus      `json:"status"`
}

func ToHeaders(m MessageMeta) (map[string]string, error) {
	var raw bytes.Buffer
	err := json.NewEncoder(&raw).Encode(&m)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	gjson.ParseBytes(raw.Bytes()).ForEach(func(key, value gjson.Result) bool {
		headers[fmt.Sprintf("X-LRJQ-%s", key.String())] = url.PathEscape(value.String())
		return true
	})
	return headers, nil
}

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
		headers, err := ToHeaders(MessageMeta{
			ID:            msg.ID,
			QueueID:       msg.QueueID,
			WorkSignature: msg.WorkSignature,
			CreatedAt:     msg.CreatedAt,
			StartedAt:     msg.StartedAt,
			Tries:         msg.Tries,
			Priority:      msg.Priority,
			Status:        msg.Status,
		})
		if err != nil {
			logger.Err(err).Send()
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		for k, v := range headers {
			rw.Header().Add(k, v)
		}
		rw.Header().Add("content-type", "application/octet-stream")
		rw.Header().Add("content-length", fmt.Sprintf("%v", len(msg.Payload)))
		if _, err := io.Copy(rw, bytes.NewReader(msg.Payload)); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
