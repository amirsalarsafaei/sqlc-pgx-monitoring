//go:build integration

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"

	"examples/db"
	"examples/db/entities/integration_test"
)

// pgConnConfig points at the throwaway Postgres started in TestMain and shared
// by every suite in the package.
var pgConnConfig db.DBConfig

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx := context.Background()

	scripts, err := filepath.Glob(filepath.Join("db", "migrations", "*.up.sql"))
	if err != nil {
		return 0, fmt.Errorf("listing migrations: %w", err)
	}
	sort.Strings(scripts)

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("example_db"),
		postgres.WithUsername("example"),
		postgres.WithPassword("complex-password"),
		postgres.WithInitScripts(scripts...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		),
	)
	if err != nil {
		return 0, fmt.Errorf("starting postgres container: %w", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminating postgres container: %v\n", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolving container host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return 0, fmt.Errorf("resolving container port: %w", err)
	}

	pgConnConfig = db.DBConfig{
		User: "example",
		Pwd:  "complex-password",
		Host: host,
		Port: port.Port(),
		Name: "example_db",
	}

	return m.Run(), nil
}

// dbSuite is the shared base for every integration suite. It owns a pool wired
// to in-memory span and metric recorders, and resets both the database and the
// recorders before each test so suites and tests stay independent.
type dbSuite struct {
	suite.Suite

	ctx          context.Context
	pool         *pgxpool.Pool
	querier      integration_test.Querier
	spanRecorder *tracetest.SpanRecorder
	metricReader *sdkmetric.ManualReader
}

func (s *dbSuite) SetupSuite() {
	s.ctx = context.Background()
	s.spanRecorder = tracetest.NewSpanRecorder()
	s.metricReader = sdkmetric.NewManualReader()

	pool, err := db.GetConnectionPool(s.ctx, pgConnConfig,
		dbtracer.WithTraceProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(s.spanRecorder))),
		dbtracer.WithMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(s.metricReader))),
		dbtracer.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		dbtracer.WithIncludeSpanNameSuffix(true),
	)
	s.Require().NoError(err)

	s.pool = pool
	s.querier = integration_test.New()
}

func (s *dbSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *dbSuite) SetupTest() {
	_, err := s.pool.Exec(s.ctx, "TRUNCATE comments, posts, users RESTART IDENTITY CASCADE")
	s.Require().NoError(err)

	s.spanRecorder.Reset()
	var rm metricdata.ResourceMetrics
	_ = s.metricReader.Collect(s.ctx, &rm)
}

func (s *dbSuite) mustCreateUser(username, email string) integration_test.User {
	s.T().Helper()
	user, err := s.querier.CreateUser(s.ctx, s.pool, username, email)
	s.Require().NoError(err)
	return user
}

// spansByName returns the recorded, ended spans whose name matches exactly.
func (s *dbSuite) spansByName(name string) []sdktrace.ReadOnlySpan {
	var matched []sdktrace.ReadOnlySpan
	for _, span := range s.spanRecorder.Ended() {
		if span.Name() == name {
			matched = append(matched, span)
		}
	}
	return matched
}

func spanAttributes(span sdktrace.ReadOnlySpan) map[attribute.Key]attribute.Value {
	attrs := span.Attributes()
	m := make(map[attribute.Key]attribute.Value, len(attrs))
	for _, a := range attrs {
		m[a.Key] = a.Value
	}
	return m
}
