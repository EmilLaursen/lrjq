package dbtest

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/EmilLaursen/lrjq/src/core"
	"github.com/jackc/pgx/v4/pgxpool"
)

var pool *pgxpool.Pool

func GetTestDbTx(t *testing.T) (core.GenericPGX, context.Context) {
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("%s", err)
	}
	t.Cleanup(func() { tx.Rollback(ctx) })
	return tx, ctx
}

func init() {
	var err error
	dbURL := os.Getenv("TEST_POSTGRES_CONN")
	ctx := context.Background()
	pool, err = pgxpool.Connect(ctx, dbURL)
	if err != nil {
		log.Printf("connect err: %s", err)
	}
}
