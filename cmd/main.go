package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/EmilLaursen/lrjq/openapi"
	queue "github.com/EmilLaursen/lrjq/src"
	"github.com/EmilLaursen/lrjq/src/http_handler"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kelseyhightower/envconfig"

	"github.com/rs/zerolog"
)

type Config struct {
	PostgresConnStr string `envconfig:"POSTGRES_CONN_STR" required:"true"`
	Port            string `envconfig:"PORT" default:"8796"`
	LogLevel        string `envconfig:"LOG_LEVEL" default:"INFO"`

	HeartbeatDeadline       time.Duration `envconfig:"HEARTBEAT_DEADLINE" default:"20m"`
	MsgMaxTries             int32         `envconfig:"MAX_TRIES" default:"3"`
	DeadletterSweepInterval time.Duration `envconfig:"DEADLETTER_SWEEP_INTERVAL" default:"3m"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	level, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		log.Panic(err)
	}

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().
		Level(level)
	ctx = logger.WithContext(ctx)

	pool, err := connectPostgres(ctx, cfg.PostgresConnStr)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	var wg sync.WaitGroup

	hb := queue.RequeueFailed(ctx, pool, cfg.HeartbeatDeadline, cfg.HeartbeatDeadline/2, cfg.HeartbeatDeadline/4, &logger)
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-hb:
				logger.Debug().Msg("requeue goroutine heartbeat")
			}
		}
	}(ctx)

	dhb := queue.MoveDeadletters(ctx, pool, cfg.MsgMaxTries, cfg.DeadletterSweepInterval, cfg.DeadletterSweepInterval/2, &logger)
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-dhb:
				logger.Debug().Msg("delete deadletter goroutine heartbeat")
			}
		}
	}(ctx)

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		<-ctx.Done()
		pool.Close()
	}(ctx)

	base := chi.NewRouter()
	base.Mount("/static", http.StripPrefix("/static", openapi.OpenAPIHandler()))
	r := chi.NewRouter()
	base.Mount("/", http_handler.QueueRouter(r, &logger, pool, cfg.MsgMaxTries))

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", cfg.Port),
		Handler: base,
	}

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		<-ctx.Done()

		ctx, cl := context.WithTimeout(context.Background(), time.Second*30)
		defer cl()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Err(err).Send()
		}
	}(ctx)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Err(err).Send()
			cancel()
		}
	}()

	logger.Info().Msg("server started")
	wg.Wait()
	logger.Info().Msg("server stopped")
}

func connectPostgres(ctx context.Context, connstr string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connstr)
	if err != nil {
		return nil, err
	}
	var pool *pgxpool.Pool
	for k := 0; k < 10; k++ {
		pool, err = pgxpool.ConnectConfig(ctx, cfg)
		if err == nil {
			return pool, nil
		}
		time.Sleep(time.Duration(k) * time.Second)
	}
	return pool, err
}
