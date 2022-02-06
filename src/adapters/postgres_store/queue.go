package postgres_store

import (
	"context"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/EmilLaursen/lrjq/src/core"
	"github.com/jackc/pgtype"
)

//go:generate pggen gen go --output-dir gen/ -query-glob query.sql -schema-glob migrations/*.up.sql

type QueueStore struct {
}

func mssageToParams(msg core.Message) gen.EnqueueParams {
	return gen.EnqueueParams{
		Payload: pgtype.JSONB{
			Bytes:  msg.Payload,
			Status: pgtype.Present,
		},
		Priority: msg.Priority,
		QueueID:  msg.QueueID,
	}

}

// TODO: check what we return now. Not all information is relevant
func enqueueRowToMessage(er gen.EnqueueRow) core.Message {
	r := gen.EnqueueRow{
		ID:            0,
		QueueID:       "",
		Payload:       pgtype.JSONB{},
		WorkSignature: pgtype.UUID{},
		CreatedAt:     pgtype.Timestamptz{},
		LastHeartbeat: pgtype.Timestamptz{},
		StartedAt:     pgtype.Timestamptz{},
		DoneAt:        pgtype.Timestamptz{},
		Tries:         0,
		Priority:      0,
		Status:        "",
	}

}

func Enqueue(ctx context.Context, db core.GenericPGX, msg core.Message) error {
	r, err := gen.NewQuerier(db).Enqueue(ctx, mssageToParams(msg))
	if err != nil {
		return err
	}
}
