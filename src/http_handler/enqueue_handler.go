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
	"github.com/dolmen-go/contextio"
	"github.com/go-chi/chi/v5"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/tidwall/sjson"
)

func parseParams(r *http.Request) (gen.EnqueueParams, error) {
	logger := zerolog.Ctx(r.Context())

	if r == nil {
		return gen.EnqueueParams{}, nil
	}

	var aerr error
	var priority int32
	priorityRaw := r.URL.Query().Get("priority")
	if priorityRaw != "" {
		i, perr := strconv.ParseInt(priorityRaw, 10, 32)
		if perr != nil {
			aerr = perr
		}
		priority = int32(i)
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
	if err := json.NewEncoder(&hds).Encode(&headers); err != nil {
		if aerr == nil {
			aerr = err
		} else {
			aerr = eris.Wrap(aerr, err.Error())
		}
	}

	bdy, err := io.ReadAll(contextio.NewReader(r.Context(), r.Body))
	if err != nil {
		if aerr == nil {
			aerr = err
		} else {
			aerr = eris.Wrap(aerr, err.Error())
		}
		logger.Err(err).Bytes("payload", bdy).Send()
	}

	return gen.EnqueueParams{
		Payload:  bdy,
		Priority: priority,
		QueueID:  chi.URLParam(r, "queueID"),
	}, aerr
}

func transformOutput(er gen.EnqueueRow) ([]byte, error) {
	er.Payload = nil
	var raw bytes.Buffer
	err := json.NewEncoder(&raw).Encode(&er)
	if err != nil {
		return nil, err
	}
	return sjson.DeleteBytes(raw.Bytes(), "payload")
}

type enqueueFunc func(ctx context.Context, msg gen.EnqueueParams) (gen.EnqueueRow, error)

func enqueueHandler(enqueue enqueueFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		params, err := parseParams(r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(rw).Encode(map[string]interface{}{
				"error": err.Error(),
			})
			if err != nil {
				logger.Err(err).Send()
			}
			return
		}

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
			err := json.NewEncoder(rw).Encode(map[string]interface{}{
				"error": "internal error",
			})
			if err != nil {
				logger.Err(err).Send()
			}
			return
		}
		rw.Header().Add("content-type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		out, err := transformOutput(msg)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Write(out)
	}
}
