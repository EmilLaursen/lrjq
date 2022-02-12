package http_handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
)

type reportDoneFunc func(ctx context.Context, id int32, workSignature pgtype.UUID) (pgconn.CommandTag, error)

func reportDoneHandler(reportDone reportDoneFunc) http.HandlerFunc {
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

		id, workSignature, err := parseHeartbeatParams(msgID, workID)
		if err != nil {
			logger.Err(err).Send()
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		cmd, err := reportDone(ctx, id, workSignature)
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

		if cmd.RowsAffected() == 0 {
			logger.Err(err).Send()
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
