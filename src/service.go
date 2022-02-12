package queue

import (
	"context"
	"time"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store"
	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/jackc/pgtype"
	"github.com/rs/zerolog"
)

func RequeueFailed(
	ctx context.Context,
	db postgres_store.GenericPGX,
	heartbeatDeadline time.Duration,
	requeueInterval time.Duration,
	heartbeatInterval time.Duration,
	logger *zerolog.Logger,
) <-chan struct{} {
	pulseCh := make(chan struct{})
	pulse := func() {
		select {
		case pulseCh <- struct{}{}:
		default:
		}
	}

	var dl pgtype.Interval
	dl.Set(&heartbeatDeadline)

	t := time.NewTicker(requeueInterval)

	go func(ctx context.Context) {
		defer t.Stop()
		defer close(pulseCh)
		for {
			select {
			case <-time.After(heartbeatInterval):
				pulse()
			case <-ctx.Done():
				return
			case <-t.C:
				cmd, err := gen.NewQuerier(db).RequeueFailed(ctx, dl)
				if err != nil {
					logger.Err(err).Msg("requeue failed error")
				} else {
					logger.Debug().Int64("requeued", cmd.RowsAffected()).Msg("requeued lost")
				}
			}
		}
	}(ctx)

	return pulseCh
}

func MoveDeadletters(
	ctx context.Context,
	db postgres_store.GenericPGX,
	maxTries int32,
	moveDeadletterInterval time.Duration,
	heartbeatInterval time.Duration,
	logger *zerolog.Logger,
) <-chan struct{} {

	pulseCh := make(chan struct{})
	pulse := func() {
		select {
		case pulseCh <- struct{}{}:
		default:
		}
	}

	t := time.NewTicker(moveDeadletterInterval)

	go func(ctx context.Context) {
		defer t.Stop()
		defer close(pulseCh)
		for {
			select {
			case <-time.After(heartbeatInterval):
				pulse()
			case <-ctx.Done():
				return
			case <-t.C:
				cmd, err := gen.NewQuerier(db).DeleteDeadLetters(ctx, maxTries)
				if err != nil {
					logger.Err(err).Msg("delete deead letters failed error")
				} else {
					logger.Debug().Int64("deleted deadletter", cmd.RowsAffected()).Msg("deleted deadletters")
				}
			}
		}
	}(ctx)

	return pulseCh
}
