package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"

	"examples/db"
	"examples/db/store"
)

const (
	scopeName  = "github.com/amirsalarsafaei/sqlc-pgx-monitoring/examples"
	maxConns   = 4
	countUsers = "-- name: CountUsers :one\nSELECT count(*) FROM users"
)

func main() {
	interval := flag.Duration("interval", time.Second, "pause between workload rounds")
	rounds := flag.Int("rounds", 0, "workload rounds to run; 0 runs until interrupted")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *interval, *rounds); err != nil {
		slog.Error("workload stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, interval time.Duration, rounds int) error {
	shutdown, err := setupTelemetry(ctx)
	if err != nil {
		return fmt.Errorf("setting up telemetry: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			slog.Error("flushing telemetry", "error", err)
		}
	}()

	connString := getEnv("DATABASE_URL", "postgres://example:complex-password@localhost:5432/example_db?sslmode=disable")
	tracer, err := dbtracer.NewDBTracer("example_db",
		dbtracer.WithLogger(otelslog.NewLogger(scopeName)),
		dbtracer.WithIncludeSpanNameSuffix(true),
		dbtracer.WithIncludeSQLText(true),
	)
	if err != nil {
		return fmt.Errorf("creating tracer: %w", err)
	}

	pool, err := db.NewPool(ctx, connString, tracer, func(c *pgxpool.Config) {
		c.MaxConns = maxConns
		c.MinConns = 1
		c.MaxConnLifetime = time.Minute
		c.MaxConnLifetimeJitter = 10 * time.Second
		c.MaxConnIdleTime = 20 * time.Second
		c.HealthCheckPeriod = 5 * time.Second
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := poolstatus.Register(pool, poolstatus.WithAttributes(semconv.DBClientConnectionsPoolName("workload"))); err != nil {
		return fmt.Errorf("registering pool status: %w", err)
	}

	w := &workload{
		pool:       pool,
		queries:    store.New(),
		tracer:     otel.Tracer(scopeName),
		dbTracer:   tracer,
		connString: connString,
	}
	slog.Info("workload started", "interval", interval, "rounds", rounds)

	return w.run(ctx, interval, rounds)
}

func setupTelemetry(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("sqlc-pgx-monitoring-example")),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}
	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}
	logExporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating log exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
		sdkmetric.WithResource(res),
	)
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)), sdklog.WithResource(res))

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	global.SetLoggerProvider(lp)

	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx), lp.Shutdown(ctx))
	}, nil
}

type scenario struct {
	name  string
	every int
	run   func(context.Context) error
}

type workload struct {
	pool       *pgxpool.Pool
	queries    *store.Queries
	tracer     trace.Tracer
	dbTracer   dbtracer.Tracer
	connString string
	seq        atomic.Int64
}

func (w *workload) run(ctx context.Context, interval time.Duration, rounds int) error {
	scenarios := []scenario{
		{name: "crud", run: w.crud},
		{name: "unique-violation", run: w.uniqueViolation},
		{name: "batch", run: w.batch},
		{name: "batch-rows", run: w.batchRows},
		{name: "failed-batch", run: w.failedBatch},
		{name: "copy-from", run: w.copyFrom},
		{name: "raw-sql", run: w.rawSQL},
		{name: "prepare", run: w.prepare},
		{name: "pool-pressure", every: 15, run: w.poolPressure},
		{name: "canceled-acquire", every: 10, run: w.canceledAcquire},
		{name: "connect-failure", every: 10, run: w.connectFailure},
		{name: "cleanup", every: 60, run: w.cleanup},
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for round := 1; rounds == 0 || round <= rounds; round++ {
		for _, sc := range scenarios {
			if sc.every > 1 && round%sc.every != 0 {
				continue
			}

			ctx, span := w.tracer.Start(ctx, "workload "+sc.name)
			if err := sc.run(ctx); err != nil {
				span.SetStatus(codes.Error, err.Error())
				slog.Error("scenario failed", "scenario", sc.name, "error", err)
			}
			span.End()
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}

	return nil
}

func (w *workload) username(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, w.seq.Add(1))
}

func (w *workload) crud(ctx context.Context) error {
	return pgx.BeginFunc(ctx, w.pool, func(tx pgx.Tx) error {
		name := w.username("crud")
		user, err := w.queries.CreateUser(ctx, tx, name, name+"@example.com")
		if err != nil {
			return err
		}
		post, err := w.queries.CreatePost(ctx, tx, user.ID, "hello", "first post")
		if err != nil {
			return err
		}
		if _, err := w.queries.CreateComment(ctx, tx, post.ID, user.ID, "nice"); err != nil {
			return err
		}
		if _, err := w.queries.GetPostWithStats(ctx, tx, post.ID); err != nil {
			return err
		}
		if _, err := w.queries.UpdateUserEmail(ctx, tx, user.ID, name+"@example.org"); err != nil {
			return err
		}
		if _, err := w.queries.GetUserByID(ctx, tx, user.ID); err != nil {
			return err
		}
		if _, err := w.queries.ListUsers(ctx, tx, 10); err != nil {
			return err
		}

		return w.queries.DeleteUser(ctx, tx, user.ID)
	})
}

func (w *workload) uniqueViolation(ctx context.Context) error {
	name := w.username("dup")
	if _, err := w.queries.CreateUser(ctx, w.pool, name, name+"@example.com"); err != nil {
		return err
	}

	_, err := w.queries.CreateUser(ctx, w.pool, name, name+"@example.org")

	return expectSQLState(err, "23505")
}

func (w *workload) batch(ctx context.Context) error {
	params := make([]store.InsertUsersParams, 3)
	for i := range params {
		name := w.username("batch")
		params[i] = store.InsertUsersParams{Username: name, Email: name + "@example.com"}
	}

	var errs []error
	results := w.queries.InsertUsers(ctx, w.pool, params)
	results.Exec(func(_ int, err error) { errs = append(errs, err) })

	return errors.Join(append(errs, results.Close())...)
}

func (w *workload) batchRows(ctx context.Context) error {
	users, err := w.queries.ListUsers(ctx, w.pool, 3)
	if err != nil {
		return err
	}

	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}

	var errs []error
	results := w.queries.GetUsersByID(ctx, w.pool, ids)
	results.QueryRow(func(_ int, _ store.User, err error) { errs = append(errs, err) })

	return errors.Join(append(errs, results.Close())...)
}

func (w *workload) failedBatch(ctx context.Context) error {
	name := w.username("failed-batch")
	results := w.queries.InsertUsers(ctx, w.pool, []store.InsertUsersParams{
		{Username: name, Email: name + "@example.com"},
		{Username: name, Email: name + "@example.org"},
		{Username: w.username("failed-batch"), Email: name + "@example.net"},
	})
	results.Exec(func(int, error) {})

	return expectSQLState(results.Close(), "23505")
}

func (w *workload) copyFrom(ctx context.Context) error {
	rows := make([]store.CopyUsersParams, 5)
	for i := range rows {
		name := w.username("copy")
		rows[i] = store.CopyUsersParams{Username: name, Email: name + "@example.com"}
	}

	_, err := w.queries.CopyUsers(ctx, w.pool, rows)

	return err
}

func (w *workload) rawSQL(ctx context.Context) error {
	_, err := w.pool.Exec(ctx, "SELECT pg_sleep(random() * 0.1)")

	return err
}

func (w *workload) prepare(ctx context.Context) error {
	return w.pool.AcquireFunc(ctx, func(c *pgxpool.Conn) error {
		_, err := c.Conn().Prepare(ctx, "count_users", countUsers)
		return err
	})
}

func (w *workload) poolPressure(ctx context.Context) error {
	var wg sync.WaitGroup
	errs := make([]error, 2*maxConns)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = w.pool.Exec(ctx, "SELECT pg_sleep(0.2)")
		}()
	}
	wg.Wait()

	return errors.Join(errs...)
}

func (w *workload) canceledAcquire(ctx context.Context) error {
	held := make([]*pgxpool.Conn, 0, maxConns)
	defer func() {
		for _, c := range held {
			c.Release()
		}
	}()

	for range maxConns {
		c, err := w.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		held = append(held, c)
	}

	acquireCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	c, err := w.pool.Acquire(acquireCtx)
	if err == nil {
		c.Release()
		return errors.New("acquire on an exhausted pool succeeded")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return nil
}

func (w *workload) connectFailure(ctx context.Context) error {
	cfg, err := pgx.ParseConfig(w.connString)
	if err != nil {
		return err
	}
	cfg.Password = "wrong-password"
	cfg.Tracer = w.dbTracer

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err == nil {
		_ = conn.Close(ctx)
		return errors.New("connect with a wrong password succeeded")
	}

	return expectSQLState(err, "28P01")
}

func (w *workload) cleanup(ctx context.Context) error {
	_, err := w.pool.Exec(ctx, "TRUNCATE comments, posts, users")

	return err
}

func expectSQLState(err error, code string) error {
	if err == nil {
		return fmt.Errorf("want SQLSTATE %s, got no error", code)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == code {
		return nil
	}

	return fmt.Errorf("want SQLSTATE %s, got %w", code, err)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
