package http_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmilLaursen/lrjq/src/adapters/postgres_store/gen"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
)

type enq struct {
	ID            int32           `json:"id"`
	QueueID       string          `json:"queue_id"`
	Payload       json.RawMessage `json:"payload"`
	WorkSignature uuid.UUID       `json:"work_signature"`
	CreatedAt     time.Time       `json:"created_at"`
	LastHeartbeat time.Time       `json:"last_heartbeat"`
	StartedAt     time.Time       `json:"started_at"`
	DoneAt        time.Time       `json:"done_at"`
	Tries         int32           `json:"tries"`
	Priority      int32           `json:"priority"`
}

func toEnqRow(e enq) gen.EnqueueRow {
	return gen.EnqueueRow{
		ID:      e.ID,
		QueueID: e.QueueID,
		Payload: pgtype.JSONB{
			Bytes:  e.Payload,
			Status: pgtype.Present,
		},
		WorkSignature: pgtype.UUID{
			Bytes:  e.WorkSignature,
			Status: pgtype.Present,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:   e.CreatedAt,
			Status: pgtype.Present,
		},
		LastHeartbeat: pgtype.Timestamptz{
			Time:   e.LastHeartbeat,
			Status: pgtype.Present,
		},
		StartedAt: pgtype.Timestamptz{
			Time:   e.StartedAt,
			Status: pgtype.Present,
		},
		DoneAt: pgtype.Timestamptz{
			Time:   e.DoneAt,
			Status: pgtype.Present,
		},
		Tries:    e.Tries,
		Priority: e.Priority,
		Status:   gen.JobStatusReady,
	}
}

func TestEnqueueSerialize(t *testing.T) {

	uid, _ := uuid.NewV4()
	n := time.Now()

	tests := []struct {
		name          string
		enqueueOutput enq
		status        int
	}{
		{"serializable", enq{
			QueueID:       "qid",
			Payload:       []byte(`{"msg":"hello"}`),
			WorkSignature: uid,
			CreatedAt:     n,
			LastHeartbeat: n,
			StartedAt:     n,
			DoneAt:        n,
			Tries:         5,
			Priority:      10,
		}, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var enqueuer enqueueFunc = func(ctx context.Context, msg gen.EnqueueParams) (gen.EnqueueRow, error) {
				return toEnqRow(tc.enqueueOutput), nil
			}

			handler := enqueueHandler(enqueuer)

			emptyReq := httptest.NewRequest(http.MethodPost, "http://does.not.matter", http.NoBody)
			rec := httptest.NewRecorder()
			handler(rec, emptyReq)

			resp := rec.Result()
			assert.Equal(t, tc.status, resp.StatusCode)
			var x map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&x)
			assert.Nil(t, err)
		})
	}
}
