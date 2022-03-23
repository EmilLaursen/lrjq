package http_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
)

type workReportFunc func(ctx context.Context, id int32, workSignature pgtype.UUID) (pgconn.CommandTag, error)

var _ workReportFunc = (&gen.DBQuerier{}).Ack
var _ workReportFunc = (&gen.DBQuerier{}).Nack
var _ workReportFunc = (&gen.DBQuerier{}).Requeue

func workReportHandler(workReport workReportFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		msgID := chi.URLParam(r, "ID")
		workID := chi.URLParam(r, "workID")

		l := logger.With().
			Str("msg_id", msgID).
			Str("work_id", workID).
			Logger()
		logger = &l
		ctx = logger.WithContext(ctx)

		id, workSignature, err := parseIDParams(msgID, workID)
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

		cmd, err := workReport(ctx, id, workSignature)
		if err != nil {
			logger.Err(err).Send()
			switch {
			case errors.Is(err, pgx.ErrNoRows) || cmd.RowsAffected() == 0:
				rw.WriteHeader(http.StatusNotFound)
			default:
				rw.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
