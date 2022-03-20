package postgres_store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/dbtest"
	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type EnqueueMessage struct {
	Payload  json.RawMessage
	Priority int32
	QueueID  string
}

type Message struct {
	EnqueueMessage

	CreatedAt     time.Time
	Tries         int
	WordSignature pgtype.UUID
}

func MessageToParams(msg EnqueueMessage) gen.EnqueueParams {
	return gen.EnqueueParams{
		Payload:  msg.Payload,
		Priority: msg.Priority,
		QueueID:  msg.QueueID,
	}
}

func TestEnqueueDequeue(t *testing.T) {

	t.Run("gives_back_payload", func(t *testing.T) {
		tx, ctx := dbtest.GetTestDbTx(t)
		queueID := "qid"
		payload := []byte(`{"msg":"hello"}`)
		row, err := gen.NewQuerier(tx).Enqueue(ctx, MessageToParams(EnqueueMessage{
			Payload: payload,
			QueueID: queueID,
		}))
		require.Nil(t, err)

		rrow, err := gen.NewQuerier(tx).Dequeue(ctx, queueID)
		require.Nil(t, err)
		assert.JSONEq(t, string(row.Payload), string(rrow.Payload))
	})

	t.Run("sets_workid", func(t *testing.T) {
		tx, ctx := dbtest.GetTestDbTx(t)
		querier := gen.NewQuerier(tx)

		queueID := "qid"
		payload := []byte(`{"msg":"hello"}`)
		row, err := querier.Enqueue(ctx, MessageToParams(EnqueueMessage{
			Payload: payload,
			QueueID: queueID,
		}))
		require.Nil(t, err)
		assert.Zero(t, row.WorkSignature.Bytes)

		rrow, err := gen.NewQuerier(tx).Dequeue(ctx, queueID)
		require.Nil(t, err)
		assert.NotZero(t, rrow.WorkSignature.Bytes)
	})

	t.Run("report_done_removes_work", func(t *testing.T) {
		tx, ctx := dbtest.GetTestDbTx(t)
		querier := gen.NewQuerier(tx)

		queueID := "qid"
		payload := []byte(`{"msg":"hello"}`)
		row, err := querier.Enqueue(ctx, MessageToParams(EnqueueMessage{
			Payload: payload,
			QueueID: queueID,
		}))
		require.Nil(t, err)
		assert.Zero(t, row.WorkSignature.Bytes)

		rrow, err := querier.Dequeue(ctx, queueID)
		require.Nil(t, err)
		assert.NotZero(t, rrow.WorkSignature.Bytes)

		cmd, err := querier.ReportDone(ctx, rrow.ID, rrow.WorkSignature)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), cmd.RowsAffected())

		_, err = querier.Dequeue(ctx, queueID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("heartbeat_sets_last_heartbeat", func(t *testing.T) {
		tx, ctx := dbtest.GetTestDbTx(t)
		querier := gen.NewQuerier(tx)

		queueID := "qid"
		payload := []byte(`{"msg":"hello"}`)
		row, err := querier.Enqueue(ctx, MessageToParams(EnqueueMessage{
			Payload: payload,
			QueueID: queueID,
		}))
		require.Nil(t, err)
		assert.Zero(t, row.WorkSignature.Bytes)

		rrow, err := querier.Dequeue(ctx, queueID)
		require.Nil(t, err)
		assert.NotZero(t, rrow.WorkSignature.Bytes)

		cmd, err := querier.SendHeartBeat(ctx, rrow.ID, rrow.WorkSignature)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), cmd.RowsAffected())
	})

	t.Run("report_done_after_timeout_fails", func(t *testing.T) {
		tx, ctx := dbtest.GetTestDbTx(t)
		querier := gen.NewQuerier(tx)

		queueID := "qid"
		payload := []byte(`{"msg":"hello"}`)
		_, err := querier.Enqueue(ctx, MessageToParams(EnqueueMessage{
			Payload: payload,
			QueueID: queueID,
		}))
		require.Nil(t, err)

		rrow, err := querier.Dequeue(ctx, queueID)
		require.Nil(t, err)

		var iv pgtype.Interval
		iv.Set(time.Millisecond * 0)
		cmd, err := querier.RequeueFailed(ctx, iv)
		require.Nil(t, err)
		assert.Equal(t, int64(1), cmd.RowsAffected())

		rdcmd, err := querier.ReportDone(ctx, rrow.ID, rrow.WorkSignature)
		require.Nil(t, err)
		assert.Equal(t, int64(0), rdcmd.RowsAffected())

		hbcmd, err := querier.SendHeartBeat(ctx, rrow.ID, rrow.WorkSignature)
		require.Nil(t, err)
		assert.Empty(t, hbcmd.RowsAffected())
	})
}

func TestDequeue(t *testing.T) {
	tx, ctx := dbtest.GetTestDbTx(t)

	t.Run("non_existing_queue_gives_ErrNoRows", func(t *testing.T) {
		row, err := gen.NewQuerier(tx).Dequeue(ctx, "notexists")
		assert.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Empty(t, row)
	})
}
