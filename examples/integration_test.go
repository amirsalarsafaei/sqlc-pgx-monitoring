//go:build integration

package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"

	"examples/db"
	"examples/db/store"
)

type IntegrationSuite struct {
	suite.Suite

	ctx       context.Context
	container *postgres.PostgresContainer
	dbConfig  db.Config
	admin     *pgxpool.Pool
	queries   *store.Queries

	spans         *tracetest.SpanRecorder
	metrics       *sdkmetric.ManualReader
	meterProvider *sdkmetric.MeterProvider
	logs          *logRecorder
	pool          *pgxpool.Pool
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

func (s *IntegrationSuite) SetupSuite() {
	s.ctx = context.Background()
	s.queries = store.New()

	migrations, err := filepath.Glob(filepath.Join("db", "migrations", "*.up.sql"))
	s.Require().NoError(err)
	sort.Strings(migrations)

	s.container, err = postgres.Run(s.ctx, "postgres:16-alpine",
		postgres.WithDatabase("example_db"),
		postgres.WithUsername("example"),
		postgres.WithPassword("complex-password"),
		postgres.WithInitScripts(migrations...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		),
	)
	s.Require().NoError(err)

	host, err := s.container.Host(s.ctx)
	s.Require().NoError(err)
	port, err := s.container.MappedPort(s.ctx, "5432/tcp")
	s.Require().NoError(err)

	s.dbConfig = db.Config{
		User:     "example",
		Password: "complex-password",
		Host:     host,
		Port:     port.Port(),
		Name:     "example_db",
	}

	s.admin, err = pgxpool.New(s.ctx, s.dbConfig.ConnString())
	s.Require().NoError(err)
}

func (s *IntegrationSuite) TearDownSuite() {
	if s.admin != nil {
		s.admin.Close()
	}
	if s.container != nil {
		s.NoError(s.container.Terminate(context.Background()))
	}
}

func (s *IntegrationSuite) SetupTest() {
	_, err := s.admin.Exec(s.ctx, "TRUNCATE comments, posts, users RESTART IDENTITY CASCADE")
	s.Require().NoError(err)

	s.spans = tracetest.NewSpanRecorder()
	s.metrics = sdkmetric.NewManualReader()
	s.meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(s.metrics))
	s.logs = &logRecorder{}
	s.pool = s.newPool(nil)
}

func (s *IntegrationSuite) TearDownTest() {
	s.pool.Close()
}

func (s *IntegrationSuite) newPool(attrs []attribute.KeyValue, configure ...func(*pgxpool.Config)) *pgxpool.Pool {
	s.T().Helper()

	pool, err := db.NewPool(s.ctx, s.dbConfig.ConnString(), s.newTracer(), configure...)
	s.Require().NoError(err)
	s.Require().NoError(poolstatus.Register(pool,
		poolstatus.WithMeterProvider(s.meterProvider),
		poolstatus.WithAttributes(attrs...),
	))
	s.T().Cleanup(pool.Close)

	return pool
}

func (s *IntegrationSuite) newTracer() dbtracer.Tracer {
	s.T().Helper()

	tracer, err := dbtracer.NewDBTracer("example_db",
		dbtracer.WithTraceProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(s.spans))),
		dbtracer.WithMeterProvider(s.meterProvider),
		dbtracer.WithLogger(slog.New(s.logs)),
		dbtracer.WithIncludeSpanNameSuffix(true),
		dbtracer.WithIncludeSQLText(true),
	)
	s.Require().NoError(err)

	return tracer
}

func (s *IntegrationSuite) TestQueryPreparesOnce() {
	_, err := s.queries.CreateUser(s.ctx, s.pool, "alice", "alice@example.com")
	s.Require().NoError(err)
	_, err = s.queries.CreateUser(s.ctx, s.pool, "bob", "bob@example.com")
	s.Require().NoError(err)

	queries := s.spansNamed("postgresql.query CreateUser")
	s.Require().Len(queries, 2)
	for _, span := range queries {
		s.Equal(codes.Ok, span.Status().Code)
		s.Equal("CreateUser", attrs(span)[semconv.DBOperationNameKey].AsString())
		s.Equal("CreateUser", attrs(span)[dbtracer.SQLCQueryNameKey].AsString())
		s.Contains(attrs(span)[semconv.DBQueryTextKey].AsString(), "-- name: CreateUser :one")
	}

	prepare := s.requireSpan("postgresql.prepare CreateUser")
	s.Equal(queries[0].SpanContext().SpanID(), prepare.Parent().SpanID())

	s.Equal(uint64(2), s.durationPoint("query", "CreateUser", "OK").Count)
	s.Equal(uint64(1), s.durationPoint("prepare", "CreateUser", "OK").Count)
}

func (s *IntegrationSuite) TestQueryWithoutSQLCHeader() {
	_, err := s.pool.Exec(s.ctx, "SELECT 1")
	s.Require().NoError(err)

	span := s.requireSpan("postgresql.query")
	s.NotContains(attrs(span), dbtracer.SQLCQueryNameKey)
	s.Equal(uint64(1), s.durationPoint("query", "", "OK").Count)
}

func (s *IntegrationSuite) TestUniqueViolation() {
	_, err := s.queries.CreateUser(s.ctx, s.pool, "alice", "alice@example.com")
	s.Require().NoError(err)
	_, err = s.queries.CreateUser(s.ctx, s.pool, "alice", "other@example.com")
	s.Require().Error(err)

	failed := s.spansNamed("postgresql.query CreateUser")[1]
	s.Equal(codes.Error, failed.Status().Code)
	s.Equal("23505", attrs(failed)[dbtracer.DBStatusCodeKey].AsString())
	s.Equal(uint64(1), s.durationPoint("query", "CreateUser", "ERROR").Count)

	errLogs := s.logs.atLevel(slog.LevelError)
	s.Require().Len(errLogs, 1)
	s.Equal(failed.SpanContext().SpanID(), errLogs[0].spanContext.SpanID())
}

func (s *IntegrationSuite) TestBatch() {
	alice, err := s.queries.CreateUser(s.ctx, s.pool, "alice", "alice@example.com")
	s.Require().NoError(err)

	tests := []struct {
		name    string
		command string
		run     func() error
	}{
		{
			name:    "InsertUsers",
			command: "batchexec",
			run: func() error {
				var errs []error
				results := s.queries.InsertUsers(s.ctx, s.pool, []store.InsertUsersParams{
					{Username: "bob", Email: "bob@example.com"},
					{Username: "carol", Email: "carol@example.com"},
				})
				results.Exec(func(_ int, err error) { errs = append(errs, err) })
				s.Equal([]error{nil, nil}, errs)
				return results.Close()
			},
		},
		{
			name:    "GetUsersByID",
			command: "batchone",
			run: func() error {
				var ids []int64
				results := s.queries.GetUsersByID(s.ctx, s.pool, []int64{alice.ID, alice.ID})
				results.QueryRow(func(_ int, u store.User, err error) {
					s.NoError(err)
					ids = append(ids, u.ID)
				})
				s.Equal([]int64{alice.ID, alice.ID}, ids)
				return results.Close()
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.spans.Reset()

			s.Require().NoError(tt.run())

			batch := s.requireSpan("postgresql.batch " + tt.name)
			s.Equal(codes.Ok, batch.Status().Code)
			s.Equal("batch", attrs(batch)[dbtracer.PGXOperationTypeKey].AsString())
			s.Equal(tt.name, attrs(batch)[semconv.DBOperationNameKey].AsString())
			s.Equal(tt.name, attrs(batch)[dbtracer.SQLCQueryNameKey].AsString())
			s.Equal(tt.command, attrs(batch)[dbtracer.SQLCQueryCommandKey].AsString())
			queries := s.spansNamed("postgresql.batch.query " + tt.name)
			s.Require().Len(queries, 2)
			for _, span := range queries {
				s.Equal(codes.Ok, span.Status().Code)
				s.Equal(batch.SpanContext().SpanID(), span.Parent().SpanID())
			}
			s.Equal(uint64(1), s.durationPoint("batch", tt.name, "OK").Count)
		})
	}
}

func (s *IntegrationSuite) TestFailedBatchEndsEveryQuerySpan() {
	results := s.queries.InsertUsers(s.ctx, s.pool, []store.InsertUsersParams{
		{Username: "alice", Email: "alice@example.com"},
		{Username: "alice", Email: "duplicate@example.com"},
		{Username: "bob", Email: "bob@example.com"},
	})
	results.Exec(func(int, error) {})
	s.Require().Error(results.Close())

	batch := s.requireSpan("postgresql.batch InsertUsers")
	s.Equal(codes.Error, batch.Status().Code)
	s.Equal("23505", attrs(batch)[dbtracer.DBStatusCodeKey].AsString())

	queries := s.spansNamed("postgresql.batch.query InsertUsers")
	s.Require().Len(queries, 3)
	s.Equal([]codes.Code{codes.Ok, codes.Error, codes.Error}, statusCodes(queries))
	s.Equal(uint64(1), s.durationPoint("batch", "InsertUsers", "ERROR").Count)
}

// An argument pgx cannot encode fails SendBatch before anything is sent; pgx then ends the batch twice.
func (s *IntegrationSuite) TestBatchEarlyError() {
	batch := &pgx.Batch{}
	batch.Queue("-- name: Unencodable :batchexec\nSELECT $1::int", make(chan int))

	s.Require().Error(s.pool.SendBatch(s.ctx, batch).Close())

	span := s.requireSpan("postgresql.batch Unencodable")
	s.Equal(codes.Error, span.Status().Code)
	s.Equal(codes.Error, s.requireSpan("postgresql.batch.query Unencodable").Status().Code)
	s.Equal(uint64(1), s.durationPoint("batch", "Unencodable", "UNKNOWN_ERROR").Count)
	s.Len(s.logs.withMessage("batch end"), 1)
}

func (s *IntegrationSuite) TestCopyFrom() {
	copied, err := s.queries.CopyUsers(s.ctx, s.pool, []store.CopyUsersParams{
		{Username: "alice", Email: "alice@example.com"},
		{Username: "bob", Email: "bob@example.com"},
		{Username: "carol", Email: "carol@example.com"},
	})
	s.Require().NoError(err)
	s.Equal(int64(3), copied)

	span := s.requireSpan("postgresql.copy_from")
	s.Equal(codes.Ok, span.Status().Code)
	s.Equal(`"users"`, attrs(span)[semconv.DBCollectionNameKey].AsString())
	s.Equal(uint64(1), s.durationPoint("copy_from", "", "OK").Count)

	copyLogs := s.logs.withMessage("copy_from")
	s.Require().Len(copyLogs, 1)
	s.Equal(int64(3), copyLogs[0].attr("rowCount"))
}

func (s *IntegrationSuite) TestPoolAcquireAndRelease() {
	for range 3 {
		_, err := s.queries.ListUsers(s.ctx, s.pool, 10)
		s.Require().NoError(err)
	}

	acquires := s.counter("pgx.pool.trace.acquire.count")
	s.Equal(acquires, s.counter("pgx.pool.trace.release.count"))
	s.Equal(int64(len(s.spansNamed("pgxpool.acquire"))), acquires)
	waits := s.histogramPoints("pgx.pool.trace.acquire.duration")
	s.Require().Len(waits, 1)
	status, _ := waits[0].Attributes.Value(dbtracer.PGXStatusKey)
	s.Equal("OK", status.AsString())
	s.Equal(uint64(acquires), waits[0].Count)
	s.GreaterOrEqual(acquires, int64(3))
}

func (s *IntegrationSuite) TestCanceledAcquire() {
	pool := s.newPool([]attribute.KeyValue{attribute.String("pool", "limited")},
		func(c *pgxpool.Config) { c.MaxConns = 1 })
	s.spans.Reset()

	held, err := pool.Acquire(s.ctx)
	s.Require().NoError(err)
	ctx, cancel := context.WithTimeout(s.ctx, 50*time.Millisecond)
	defer cancel()
	_, err = pool.Acquire(ctx)
	s.Require().ErrorIs(err, context.DeadlineExceeded)
	held.Release()

	spans := s.spansNamed("pgxpool.acquire")
	s.Require().Len(spans, 2)
	s.Equal([]codes.Code{codes.Ok, codes.Error}, statusCodes(spans))

	obs := s.poolObservations()
	s.Equal(float64(1), obs["pgx.pool.canceled_acquires{pool=limited}"])
	s.Equal(float64(1), obs["db.client.connection.max{pool=limited}"])
}

func (s *IntegrationSuite) TestPoolStatusMirrorsPool() {
	for range 3 {
		_, err := s.queries.ListUsers(s.ctx, s.pool, 10)
		s.Require().NoError(err)
	}

	stat := s.pool.Stat()
	obs := s.poolObservations()
	s.Equal(float64(stat.AcquiredConns()), obs["db.client.connections.usage{state=used}"])
	s.Equal(float64(stat.IdleConns()), obs["db.client.connections.usage{state=idle}"])
	s.Equal(float64(stat.MaxConns()), obs["db.client.connection.max"])
	s.Equal(float64(stat.AcquireCount()), obs["pgx.pool.acquires"])
	s.Equal(float64(stat.NewConnsCount()), obs["pgx.pool.connections.created"])
}

func (s *IntegrationSuite) TestConnectFailure() {
	cfg := s.dbConfig
	cfg.Password = "wrong-password"
	s.spans.Reset()

	_, err := db.NewPool(s.ctx, cfg.ConnString(), s.newTracer())
	s.Require().Error(err)

	connect := s.requireSpan("postgresql.connect")
	s.Equal(codes.Error, connect.Status().Code)
	s.Equal("28P01", attrs(connect)[dbtracer.DBStatusCodeKey].AsString())
	s.Equal(uint64(1), s.durationPoint("connect", "", "FATAL").Count)
}

func (s *IntegrationSuite) spansNamed(name string) []sdktrace.ReadOnlySpan {
	var spans []sdktrace.ReadOnlySpan
	for _, span := range s.spans.Ended() {
		if span.Name() == name {
			spans = append(spans, span)
		}
	}

	return spans
}

func (s *IntegrationSuite) requireSpan(name string) sdktrace.ReadOnlySpan {
	s.T().Helper()

	spans := s.spansNamed(name)
	s.Require().Len(spans, 1, "spans named %q", name)

	return spans[0]
}

func (s *IntegrationSuite) collect() []metricdata.Metrics {
	s.T().Helper()

	var rm metricdata.ResourceMetrics
	s.Require().NoError(s.metrics.Collect(s.ctx, &rm))

	var metrics []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		metrics = append(metrics, sm.Metrics...)
	}

	return metrics
}

func (s *IntegrationSuite) histogramPoints(name string) []metricdata.HistogramDataPoint[float64] {
	for _, m := range s.collect() {
		if hist, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == name {
			return hist.DataPoints
		}
	}

	return nil
}

func (s *IntegrationSuite) durationPoint(operation, name, status string) metricdata.HistogramDataPoint[float64] {
	s.T().Helper()

	for _, point := range s.histogramPoints(semconv.DBClientOperationDurationName) {
		op, _ := point.Attributes.Value(dbtracer.PGXOperationTypeKey)
		queryName, _ := point.Attributes.Value(dbtracer.SQLCQueryNameKey)
		pgxStatus, _ := point.Attributes.Value(dbtracer.PGXStatusKey)
		if op.AsString() == operation && queryName.AsString() == name && pgxStatus.AsString() == status {
			return point
		}
	}
	s.FailNow("duration point not found", "operation=%s name=%s status=%s", operation, name, status)

	return metricdata.HistogramDataPoint[float64]{}
}

func (s *IntegrationSuite) counter(name string) int64 {
	s.T().Helper()

	for _, m := range s.collect() {
		if sum, ok := m.Data.(metricdata.Sum[int64]); ok && m.Name == name {
			s.Require().Len(sum.DataPoints, 1)
			return sum.DataPoints[0].Value
		}
	}
	s.FailNow("counter not found", name)

	return 0
}

func (s *IntegrationSuite) poolObservations() map[string]float64 {
	obs := map[string]float64{}
	add := func(name string, set attribute.Set, value float64) {
		if enc := set.Encoded(attribute.DefaultEncoder()); enc != "" {
			name += "{" + enc + "}"
		}
		obs[name] = value
	}

	for _, m := range s.collect() {
		switch data := m.Data.(type) {
		case metricdata.Gauge[int64]:
			for _, p := range data.DataPoints {
				add(m.Name, p.Attributes, float64(p.Value))
			}
		case metricdata.Sum[int64]:
			for _, p := range data.DataPoints {
				add(m.Name, p.Attributes, float64(p.Value))
			}
		case metricdata.Sum[float64]:
			for _, p := range data.DataPoints {
				add(m.Name, p.Attributes, p.Value)
			}
		}
	}

	return obs
}

func attrs(span sdktrace.ReadOnlySpan) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value)
	for _, a := range span.Attributes() {
		m[a.Key] = a.Value
	}

	return m
}

func statusCodes(spans []sdktrace.ReadOnlySpan) []codes.Code {
	statuses := make([]codes.Code, 0, len(spans))
	for _, span := range spans {
		statuses = append(statuses, span.Status().Code)
	}

	return statuses
}

type logEntry struct {
	record      slog.Record
	spanContext trace.SpanContext
}

func (e logEntry) attr(key string) any {
	var value any
	e.record.Attrs(func(a slog.Attr) bool {
		if a.Key != key {
			return true
		}
		value = a.Value.Any()
		return false
	})

	return value
}

type logRecorder struct {
	mu      sync.Mutex
	entries []logEntry
}

func (h *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (h *logRecorder) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *logRecorder) WithGroup(string) slog.Handler            { return h }

func (h *logRecorder) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, logEntry{record: r.Clone(), spanContext: trace.SpanContextFromContext(ctx)})

	return nil
}

func (h *logRecorder) filter(keep func(slog.Record) bool) []logEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	var entries []logEntry
	for _, e := range h.entries {
		if keep(e.record) {
			entries = append(entries, e)
		}
	}

	return entries
}

func (h *logRecorder) atLevel(level slog.Level) []logEntry {
	return h.filter(func(r slog.Record) bool { return r.Level == level })
}

func (h *logRecorder) withMessage(msg string) []logEntry {
	return h.filter(func(r slog.Record) bool { return r.Message == msg })
}
