package http_handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"

	"github.com/gofrs/uuid"
)

func parseHeartbeatParams(msgID, workID string) (int32, pgtype.UUID, error) {

	var uid pgtype.UUID
	var mid int32
	x, err := strconv.ParseInt(msgID, 10, 32)
	if err != nil {
		return mid, uid, err
	}
	mid = int32(x)

	y, err := uuid.FromString(workID)
	if err != nil {
		return mid, uid, err
	}
	uid = pgtype.UUID{
		Bytes:  y,
		Status: pgtype.Present,
	}

	return mid, uid, nil
}

type heartbeatFunc func(
	ctx context.Context,
	id int32,
	workSignature pgtype.UUID) (pgconn.CommandTag, error)

func heartbeatHandler(heartbeat heartbeatFunc) http.HandlerFunc {
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

		cmd, err := heartbeat(ctx, id, workSignature)
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
