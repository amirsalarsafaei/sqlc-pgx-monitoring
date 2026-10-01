package dbtracer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	testDBName    = "test_db"
	getUserSQL    = "-- name: GetUser :one\nSELECT id, name FROM users WHERE id = $1"
	insertUserSQL = "-- name: InsertUser :batchexec\nINSERT INTO users (name) VALUES ($1)"
	rawSQL        = "SELECT 1"
)

var (
	dbAttrs = []attribute.KeyValue{
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(testDBName),
	}
	getUserAttrs = []attribute.KeyValue{
		SQLCQueryNameKey.String("GetUser"),
		SQLCQueryCommandKey.String("one"),
	}
	insertUserAttrs = []attribute.KeyValue{
		SQLCQueryNameKey.String("InsertUser"),
		SQLCQueryCommandKey.String("batchexec"),
	}
	errUniqueViolation = fmt.Errorf("insert user: %w", &pgconn.PgError{
		Severity: "ERROR",
		Code:     "23505",
		Message:  "duplicate key value violates unique constraint",
	})
	errAdminShutdown = &pgconn.PgError{
		Severity: "FATAL",
		Code:     "57P01",
		Message:  "terminating connection due to administrator command",
	}
	errConnClosed = errors.New("conn closed")
)

type DBTracerSuite struct {
	suite.Suite

	ctx            context.Context
	spans          *tracetest.SpanRecorder
	tracerProvider trace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	metrics        *sdkmetric.ManualReader
	logs           *logRecorder
	tracer         Tracer
}

func TestDBTracerSuite(t *testing.T) {
	suite.Run(t, new(DBTracerSuite))
}

func (s *DBTracerSuite) SetupTest() {
	s.ctx = context.Background()
	s.spans = tracetest.NewSpanRecorder()
	s.tracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(s.spans))
	s.metrics = sdkmetric.NewManualReader()
	s.meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(s.metrics))
	s.logs = &logRecorder{}
	s.tracer = s.newTracer()
}

func (s *DBTracerSuite) newTracer(opts ...Option) Tracer {
	s.T().Helper()

	tracer, err := NewDBTracer(testDBName, append([]Option{
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithLogger(slog.New(s.logs)),
	}, opts...)...)
	s.Require().NoError(err)

	return tracer
}

func (s *DBTracerSuite) TestNewDBTracerRejectsEmptyDatabaseName() {
	tracer, err := NewDBTracer("")

	s.ErrorIs(err, ErrDatabaseNameEmpty)
	s.Nil(tracer)
}

func (s *DBTracerSuite) TestQuery() {
	tests := []struct {
		name        string
		sql         string
		err         error
		spanAttrs   []attribute.KeyValue
		metricAttrs []attribute.KeyValue
		status      codes.Code
	}{
		{
			name:        "sqlc query succeeds",
			sql:         getUserSQL,
			spanAttrs:   []attribute.KeyValue{pgxOperationQuery, semconv.DBOperationName("GetUser")},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("OK")},
			status:      codes.Ok,
		},
		{
			name: "postgres error",
			sql:  getUserSQL,
			err:  errUniqueViolation,
			spanAttrs: []attribute.KeyValue{
				pgxOperationQuery, semconv.DBOperationName("GetUser"), DBStatusCodeKey.String("23505"),
			},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("ERROR")},
			status:      codes.Error,
		},
		{
			name:        "driver error",
			sql:         getUserSQL,
			err:         errConnClosed,
			spanAttrs:   []attribute.KeyValue{pgxOperationQuery, semconv.DBOperationName("GetUser")},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("UNKNOWN_ERROR")},
			status:      codes.Error,
		},
		{
			name:        "no rows",
			sql:         getUserSQL,
			err:         pgx.ErrNoRows,
			spanAttrs:   []attribute.KeyValue{pgxOperationQuery, semconv.DBOperationName("GetUser")},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("UNKNOWN_ERROR")},
			status:      codes.Error,
		},
		{
			name: "fatal postgres error",
			sql:  getUserSQL,
			err:  errAdminShutdown,
			spanAttrs: []attribute.KeyValue{
				pgxOperationQuery, semconv.DBOperationName("GetUser"), DBStatusCodeKey.String("57P01"),
			},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("FATAL")},
			status:      codes.Error,
		},
		{
			name:        "query without sqlc header",
			sql:         rawSQL,
			spanAttrs:   []attribute.KeyValue{pgxOperationQuery},
			metricAttrs: []attribute.KeyValue{pgxOperationQuery, PGXStatusKey.String("OK")},
			status:      codes.Ok,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			sqlcAttrs := getUserAttrs
			if tt.sql == rawSQL {
				sqlcAttrs = nil
			}

			s.query(s.tracer, tt.sql, tt.err)

			span := s.requireSpan("postgresql.query")
			s.Equal(trace.SpanKindClient, span.SpanKind())
			s.Equal(tt.status, span.Status().Code)
			s.True(span.EndTime().After(span.StartTime()))
			s.assertAttributes(concat(dbAttrs, sqlcAttrs, tt.spanAttrs), attribute.NewSet(span.Attributes()...))
			if tt.err != nil {
				s.Equal(tt.err.Error(), span.Status().Description)
				s.Require().Len(span.Events(), 1)
				s.Equal("exception", span.Events()[0].Name)
			}

			point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
			s.Equal(uint64(1), point.Count)
			s.Positive(point.Sum)
			s.assertAttributes(concat(dbAttrs, sqlcAttrs, tt.metricAttrs), point.Attributes)
		})
	}
}

func (s *DBTracerSuite) TestQueryStartLeavesSpanOpen() {
	ctx := s.tracer.TraceQueryStart(s.ctx, nil, pgx.TraceQueryStartData{SQL: getUserSQL, Args: []any{1}})

	s.Empty(s.spans.Ended())
	span, ok := trace.SpanFromContext(ctx).(sdktrace.ReadOnlySpan)
	s.Require().True(ok)
	s.Equal("postgresql.query", span.Name())
	s.assertAttributes(concat(dbAttrs, getUserAttrs, []attribute.KeyValue{
		pgxOperationQuery, semconv.DBOperationName("GetUser"),
	}), attribute.NewSet(span.Attributes()...))

	s.tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	s.Len(s.spans.Ended(), 1)
}

func (s *DBTracerSuite) TestPrepare() {
	tests := []struct {
		name            string
		end             pgx.TracePrepareEndData
		status          codes.Code
		pgx             string
		level           slog.Level
		alreadyPrepared any
	}{
		{
			name:            "already prepared",
			end:             pgx.TracePrepareEndData{AlreadyPrepared: true},
			status:          codes.Ok,
			pgx:             "OK",
			level:           slog.LevelInfo,
			alreadyPrepared: true,
		},
		{
			name:            "newly prepared",
			end:             pgx.TracePrepareEndData{AlreadyPrepared: false},
			status:          codes.Ok,
			pgx:             "OK",
			level:           slog.LevelInfo,
			alreadyPrepared: false,
		},
		{
			name:   "fails",
			end:    pgx.TracePrepareEndData{Err: errConnClosed},
			status: codes.Error,
			pgx:    "UNKNOWN_ERROR",
			level:  slog.LevelError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			ctx := s.tracer.TracePrepareStart(s.ctx, nil, pgx.TracePrepareStartData{Name: "stmt_get_user", SQL: getUserSQL})
			s.tracer.TracePrepareEnd(ctx, nil, tt.end)

			span := s.requireSpan("postgresql.prepare")
			s.Equal(tt.status, span.Status().Code)
			s.True(span.EndTime().After(span.StartTime()))
			s.assertAttributes(concat(dbAttrs, getUserAttrs, []attribute.KeyValue{
				pgxOperationPrepare, PGXPrepareStmtNameKey.String("stmt_get_user"), semconv.DBOperationName("GetUser"),
			}), attribute.NewSet(span.Attributes()...))
			if tt.end.Err != nil {
				s.Equal(tt.end.Err.Error(), span.Status().Description)
				s.Require().Len(span.Events(), 1)
				s.Equal("exception", span.Events()[0].Name)
			}

			point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
			s.Equal(uint64(1), point.Count)
			s.Positive(point.Sum)
			s.assertAttributes(concat(dbAttrs, getUserAttrs, []attribute.KeyValue{
				pgxOperationPrepare, PGXStatusKey.String(tt.pgx),
			}), point.Attributes)

			record := s.logs.requireOne(s.T())
			s.Equal("prepare", record.Message)
			s.Equal(tt.level, record.Level)
			s.Equal(tt.alreadyPrepared, logAttr(record, "alreadyPrepared"))
			s.Equal(uint64(0), logAttr(record, "pid"))
		})
	}
}

func (s *DBTracerSuite) TestBatch() {
	ctx := s.tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{Batch: batchOf(insertUserSQL, insertUserSQL)})
	s.tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL, Args: []any{"alice"}})
	s.tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL, Args: []any{"alice"}, Err: errUniqueViolation})
	s.tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: errUniqueViolation})

	ended := s.spans.Ended()
	s.Require().Len(ended, 3)
	s.Equal("postgresql.batch", ended[2].Name())

	batchSpan := s.requireSpan("postgresql.batch")
	s.Equal(codes.Error, batchSpan.Status().Code)
	s.True(batchSpan.EndTime().After(batchSpan.StartTime()))
	s.assertAttributes(concat(dbAttrs, insertUserAttrs, []attribute.KeyValue{
		pgxOperationBatch, semconv.DBOperationName("InsertUser"), DBStatusCodeKey.String("23505"),
	}), attribute.NewSet(batchSpan.Attributes()...))

	querySpans := s.spansNamed("postgresql.batch.query")
	s.Require().Len(querySpans, 2)
	for _, span := range querySpans {
		s.Equal(batchSpan.SpanContext().SpanID(), span.Parent().SpanID())
	}
	s.Equal(codes.Ok, querySpans[0].Status().Code)
	s.assertAttributes(concat(dbAttrs, insertUserAttrs, []attribute.KeyValue{
		pgxOperationBatchQuery, semconv.DBOperationName("InsertUser"),
	}), attribute.NewSet(querySpans[0].Attributes()...))
	s.Equal(codes.Error, querySpans[1].Status().Code)
	s.assertAttributes(concat(dbAttrs, insertUserAttrs, []attribute.KeyValue{
		pgxOperationBatchQuery, semconv.DBOperationName("InsertUser"), DBStatusCodeKey.String("23505"),
	}), attribute.NewSet(querySpans[1].Attributes()...))

	point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
	s.Equal(uint64(1), point.Count)
	s.Positive(point.Sum)
	s.assertAttributes(concat(dbAttrs, insertUserAttrs, []attribute.KeyValue{
		pgxOperationBatch, PGXStatusKey.String("ERROR"),
	}), point.Attributes)
}

// pgx reports no further queued query once one fails, and none when sending the batch fails.
func (s *DBTracerSuite) TestBatchEndEndsUnreportedQuerySpans() {
	tests := []struct {
		name       string
		reported   []error
		endErr     error
		pendingErr error
	}{
		{name: "first query fails", reported: []error{errUniqueViolation}, endErr: errUniqueViolation, pendingErr: errUniqueViolation},
		{name: "send fails", endErr: errConnClosed, pendingErr: errConnClosed},
		{name: "batch ends without error", reported: []error{nil}, pendingErr: errBatchQueryNotExecuted},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			ctx := s.tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{
				Batch: batchOf(insertUserSQL, insertUserSQL, insertUserSQL),
			})
			for _, err := range tt.reported {
				s.tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL, Err: err})
			}
			s.tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: tt.endErr})

			querySpans := s.spansNamed("postgresql.batch.query")
			s.Require().Len(querySpans, 3)
			for _, span := range querySpans[len(tt.reported):] {
				s.Equal(codes.Error, span.Status().Code)
				s.Equal(tt.pendingErr.Error(), span.Status().Description)
			}
			s.Len(s.spansNamed("postgresql.batch"), 1)
		})
	}
}

// pgx ends a batch from SendBatch and again from Close when sending fails.
func (s *DBTracerSuite) TestBatchEndRecordsOnce() {
	ctx := s.tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{Batch: batchOf(insertUserSQL)})
	s.tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: errConnClosed})
	s.tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{Err: errConnClosed})

	s.Len(s.spansNamed("postgresql.batch"), 1)
	s.Equal(uint64(1), s.requireHistogramPoint(semconv.DBClientOperationDurationName).Count)
	s.Equal([]string{"batch end"}, s.logs.messages())
}

func (s *DBTracerSuite) TestBatchIgnoresQueriesBeyondQueue() {
	ctx := s.tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{})
	s.tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL})
	s.tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{})

	s.Len(s.spans.Ended(), 1)
	span := s.requireSpan("postgresql.batch")
	s.Equal(codes.Ok, span.Status().Code)
	s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxOperationBatch}), attribute.NewSet(span.Attributes()...))

	point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
	s.Positive(point.Sum)
	s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxOperationBatch, PGXStatusKey.String("OK")}), point.Attributes)
}

func (s *DBTracerSuite) TestSpanNameSuffix() {
	tracer := s.newTracer(WithIncludeSpanNameSuffix(true))

	s.query(tracer, getUserSQL, nil)
	s.query(tracer, rawSQL, nil)
	ctx := tracer.TracePrepareStart(s.ctx, nil, pgx.TracePrepareStartData{SQL: getUserSQL})
	tracer.TracePrepareEnd(ctx, nil, pgx.TracePrepareEndData{})
	ctx = tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{Batch: batchOf(insertUserSQL)})
	tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL})
	tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{})

	s.ElementsMatch([]string{
		"postgresql.query GetUser",
		"postgresql.query",
		"postgresql.prepare GetUser",
		"postgresql.batch.query InsertUser",
		"postgresql.batch InsertUser",
	}, spanNames(s.spans.Ended()))
}

func (s *DBTracerSuite) TestIncludeSQLText() {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			tracer := s.newTracer(WithIncludeSQLText(tt.enabled))

			s.query(tracer, getUserSQL, nil)
			ctx := tracer.TracePrepareStart(s.ctx, nil, pgx.TracePrepareStartData{SQL: getUserSQL})
			tracer.TracePrepareEnd(ctx, nil, pgx.TracePrepareEndData{})
			ctx = tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{Batch: batchOf(insertUserSQL)})
			tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL})
			tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{})

			want := map[string]string{
				"postgresql.query":       getUserSQL,
				"postgresql.prepare":     getUserSQL,
				"postgresql.batch.query": insertUserSQL,
				"postgresql.batch":       "",
			}
			for name, sql := range want {
				attrs := attribute.NewSet(s.requireSpan(name).Attributes()...)
				text, ok := attrs.Value(semconv.DBQueryTextKey)
				s.Equal(tt.enabled && sql != "", ok, name)
				if ok {
					s.Equal(sql, text.AsString(), name)
				}
			}
		})
	}
}

func (s *DBTracerSuite) TestConnect() {
	tests := []struct {
		name   string
		err    error
		status codes.Code
		pgx    string
		level  slog.Level
	}{
		{name: "succeeds", status: codes.Ok, pgx: "OK", level: slog.LevelInfo},
		{name: "fails", err: errConnClosed, status: codes.Error, pgx: "UNKNOWN_ERROR", level: slog.LevelError},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			ctx := s.tracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
				ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{Host: "db.local", Port: 5432, Database: testDBName}},
			})
			s.tracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{Err: tt.err})

			span := s.requireSpan("postgresql.connect")
			s.Equal(tt.status, span.Status().Code)
			s.True(span.EndTime().After(span.StartTime()))
			s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxOperationConnect}), attribute.NewSet(span.Attributes()...))
			if tt.err != nil {
				s.Equal(tt.err.Error(), span.Status().Description)
				s.Require().Len(span.Events(), 1)
				s.Equal("exception", span.Events()[0].Name)
			}

			point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
			s.Equal(uint64(1), point.Count)
			s.Positive(point.Sum)
			s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxOperationConnect, PGXStatusKey.String(tt.pgx)}), point.Attributes)

			record := s.logs.requireOne(s.T())
			s.Equal(tt.level, record.Level)
			s.Equal("db.local", logAttr(record, "host"))
			s.Equal(uint64(5432), logAttr(record, "port"))
		})
	}
}

func (s *DBTracerSuite) TestCopyFrom() {
	tests := []struct {
		name     string
		end      pgx.TraceCopyFromEndData
		status   codes.Code
		pgx      string
		level    slog.Level
		rowCount any
	}{
		{
			name:     "succeeds",
			end:      pgx.TraceCopyFromEndData{CommandTag: pgconn.NewCommandTag("COPY 3")},
			status:   codes.Ok,
			pgx:      "OK",
			level:    slog.LevelInfo,
			rowCount: int64(3),
		},
		{
			name:   "fails",
			end:    pgx.TraceCopyFromEndData{Err: errConnClosed},
			status: codes.Error,
			pgx:    "UNKNOWN_ERROR",
			level:  slog.LevelError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			ctx := s.tracer.TraceCopyFromStart(s.ctx, nil, pgx.TraceCopyFromStartData{
				TableName:   pgx.Identifier{"public", "users"},
				ColumnNames: []string{"id", "name"},
			})
			s.tracer.TraceCopyFromEnd(ctx, nil, tt.end)

			span := s.requireSpan("postgresql.copy_from")
			s.Equal(tt.status, span.Status().Code)
			s.True(span.EndTime().After(span.StartTime()))
			s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{
				pgxOperationCopyFrom, semconv.DBCollectionName(`"public"."users"`),
			}), attribute.NewSet(span.Attributes()...))
			if tt.end.Err != nil {
				s.Equal(tt.end.Err.Error(), span.Status().Description)
				s.Require().Len(span.Events(), 1)
				s.Equal("exception", span.Events()[0].Name)
			}

			point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
			s.Equal(uint64(1), point.Count)
			s.Positive(point.Sum)
			s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxOperationCopyFrom, PGXStatusKey.String(tt.pgx)}), point.Attributes)

			record := s.logs.requireOne(s.T())
			s.Equal(tt.level, record.Level)
			s.Equal(tt.rowCount, logAttr(record, "rowCount"))
		})
	}
}

func (s *DBTracerSuite) TestPoolAcquireAndRelease() {
	ctx := s.tracer.TraceAcquireStart(s.ctx, nil, pgxpool.TraceAcquireStartData{})
	s.tracer.TraceAcquireEnd(ctx, nil, pgxpool.TraceAcquireEndData{})
	ctx = s.tracer.TraceAcquireStart(s.ctx, nil, pgxpool.TraceAcquireStartData{})
	s.tracer.TraceAcquireEnd(ctx, nil, pgxpool.TraceAcquireEndData{Err: context.DeadlineExceeded})
	s.tracer.TraceRelease(nil, pgxpool.TraceReleaseData{})

	spans := s.spansNamed("pgxpool.acquire")
	s.Require().Len(spans, 2)
	s.Equal(codes.Ok, spans[0].Status().Code)
	s.Equal(codes.Error, spans[1].Status().Code)
	s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxPoolConnOperationAcquire}), attribute.NewSet(spans[0].Attributes()...))

	acquires := s.counterPoints("pgx.pool.trace.acquire.count")
	s.Require().Len(acquires, 2)
	waits := s.histogramPoints("pgx.pool.trace.acquire.duration")
	s.Require().Len(waits, 2)
	for i, status := range []string{"OK", "UNKNOWN_ERROR"} {
		statusAttr := PGXStatusKey.String(status)
		s.Equal(int64(1), acquires[i].Value)
		s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxPoolConnOperationAcquire, statusAttr}), acquires[i].Attributes)
		s.Equal(uint64(1), waits[i].Count)
		s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{statusAttr}), waits[i].Attributes)
	}

	releases := s.requireCounterPoint("pgx.pool.trace.release.count")
	s.Equal(int64(1), releases.Value)
	s.assertAttributes(concat(dbAttrs, []attribute.KeyValue{pgxPoolConnOperationReleased}), releases.Attributes)
}

func (s *DBTracerSuite) TestNonRecordingTracerProvider() {
	s.tracerProvider = noop.NewTracerProvider()
	tracer := s.newTracer()

	s.query(tracer, getUserSQL, nil)
	ctx := tracer.TracePrepareStart(s.ctx, nil, pgx.TracePrepareStartData{SQL: getUserSQL})
	tracer.TracePrepareEnd(ctx, nil, pgx.TracePrepareEndData{})
	ctx = tracer.TraceBatchStart(s.ctx, nil, pgx.TraceBatchStartData{Batch: batchOf(insertUserSQL)})
	tracer.TraceBatchQuery(ctx, nil, pgx.TraceBatchQueryData{SQL: insertUserSQL})
	tracer.TraceBatchEnd(ctx, nil, pgx.TraceBatchEndData{})
	ctx = tracer.TraceCopyFromStart(s.ctx, nil, pgx.TraceCopyFromStartData{TableName: pgx.Identifier{"users"}})
	tracer.TraceCopyFromEnd(ctx, nil, pgx.TraceCopyFromEndData{})
	ctx = tracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{ConnConfig: &pgx.ConnConfig{}})
	tracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{})
	ctx = tracer.TraceAcquireStart(s.ctx, nil, pgxpool.TraceAcquireStartData{})
	tracer.TraceAcquireEnd(ctx, nil, pgxpool.TraceAcquireEndData{})

	var operations []string
	for _, point := range s.histogramPoints(semconv.DBClientOperationDurationName) {
		op, _ := point.Attributes.Value(PGXOperationTypeKey)
		operations = append(operations, op.AsString())
	}
	s.ElementsMatch([]string{"query", "prepare", "batch", "copy_from", "connect"}, operations)
	s.Len(s.histogramPoints("pgx.pool.trace.acquire.duration"), 1)
	s.ElementsMatch([]string{
		"query", "prepare", "batch query", "batch end", "copy_from", "database connect", "acquire connection",
	}, s.logs.messages())
}

func (s *DBTracerSuite) TestEndHooksWithoutStartRecordNothing() {
	s.NotPanics(func() {
		s.tracer.TraceQueryEnd(s.ctx, nil, pgx.TraceQueryEndData{})
		s.tracer.TraceBatchQuery(s.ctx, nil, pgx.TraceBatchQueryData{})
		s.tracer.TraceBatchEnd(s.ctx, nil, pgx.TraceBatchEndData{})
		s.tracer.TraceConnectEnd(s.ctx, pgx.TraceConnectEndData{})
		s.tracer.TraceCopyFromEnd(s.ctx, nil, pgx.TraceCopyFromEndData{})
		s.tracer.TracePrepareEnd(s.ctx, nil, pgx.TracePrepareEndData{})
		s.tracer.TraceAcquireEnd(s.ctx, nil, pgxpool.TraceAcquireEndData{})
	})

	s.Empty(s.spans.Ended())
	s.Empty(s.collect())
	s.Empty(s.logs.messages())
}

func (s *DBTracerSuite) TestDefaultLogger() {
	previous := slog.Default()
	defer slog.SetDefault(previous)
	slog.SetDefault(slog.New(s.logs))

	tracer, err := NewDBTracer(testDBName, WithTraceProvider(s.tracerProvider), WithMeterProvider(s.meterProvider))
	s.Require().NoError(err)

	s.query(tracer, getUserSQL, nil)

	s.Equal([]string{"query"}, s.logs.messages())
}

func (s *DBTracerSuite) TestAllOptions() {
	tracer, err := NewDBTracer(testDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithLogger(slog.New(s.logs)),
		WithShouldLog(errorsOnly),
		WithLogArgs(false),
		WithLogArgsLenLimit(128),
		WithIncludeSQLText(true),
		WithIncludeSpanNameSuffix(true),
		WithLatencyHistogramConfig("custom.duration", "ms", "Custom duration metric"),
	)
	s.Require().NoError(err)

	s.query(tracer, getUserSQL, nil)
	s.query(tracer, getUserSQL, errConnClosed)

	spans := s.spansNamed("postgresql.query GetUser")
	s.Require().Len(spans, 2)
	for _, span := range spans {
		attrs := attribute.NewSet(span.Attributes()...)
		text, ok := attrs.Value(semconv.DBQueryTextKey)
		s.True(ok)
		s.Equal(getUserSQL, text.AsString())
	}

	m := s.requireMetric("custom.duration")
	s.Equal("ms", m.Unit)
	s.Equal("Custom duration metric", m.Description)

	record := s.logs.requireOne(s.T())
	s.Equal(slog.LevelError, record.Level)
	s.Nil(logAttr(record, "args"))
}

func (s *DBTracerSuite) TestShouldLog() {
	tests := []struct {
		name      string
		shouldLog ShouldLog
		err       error
		level     *slog.Level
	}{
		{name: "default logs success at info", err: nil, level: ptr(slog.LevelInfo)},
		{name: "default logs failure at error", err: errConnClosed, level: ptr(slog.LevelError)},
		{name: "errors only skips success", shouldLog: errorsOnly, err: nil},
		{name: "errors only logs failure", shouldLog: errorsOnly, err: errConnClosed, level: ptr(slog.LevelError)},
		{name: "filter receives the error", shouldLog: func(err error) bool { return !errors.Is(err, pgx.ErrNoRows) }, err: pgx.ErrNoRows},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			var opts []Option
			if tt.shouldLog != nil {
				opts = append(opts, WithShouldLog(tt.shouldLog))
			}

			s.query(s.newTracer(opts...), getUserSQL, tt.err)

			if tt.level == nil {
				s.Empty(s.logs.messages())
				return
			}
			record := s.logs.requireOne(s.T())
			s.Equal(*tt.level, record.Level)
			s.Equal("GetUser", logAttr(record, "query_name"))
			if tt.err != nil {
				s.Equal(tt.err.Error(), logAttr(record, "error"))
			}
		})
	}
}

func (s *DBTracerSuite) TestLogArgs() {
	args := []any{"héllo", []byte{0xde, 0xad, 0xbe, 0xef}, []byte{0x01}, 42, nil}

	tests := []struct {
		name string
		opts []Option
		want any
	}{
		{
			name: "truncated on rune boundary",
			opts: []Option{WithLogArgsLenLimit(2)},
			want: []any{"hé (truncated 3 bytes)", "dead (truncated 2 bytes)", "01", 42, nil},
		},
		{
			name: "non-positive limit uses default",
			opts: []Option{WithLogArgsLenLimit(-1)},
			want: []any{"héllo", "deadbeef", "01", 42, nil},
		},
		{
			name: "disabled",
			opts: []Option{WithLogArgs(false)},
			want: []any(nil),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			tracer := s.newTracer(tt.opts...)

			ctx := tracer.TraceQueryStart(s.ctx, nil, pgx.TraceQueryStartData{SQL: getUserSQL, Args: args})
			tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

			s.Equal(tt.want, logAttr(s.logs.requireOne(s.T()), "args"))
		})
	}
}

func (s *DBTracerSuite) TestLatencyHistogramConfig() {
	tests := []struct {
		name        string
		opts        []Option
		metricName  string
		unit        string
		description string
		bounds      []float64
	}{
		{
			name:        "default",
			metricName:  semconv.DBClientOperationDurationName,
			unit:        semconv.DBClientOperationDurationUnit,
			description: semconv.DBClientOperationDurationDescription,
			bounds:      defaultBucketBoundaries,
		},
		{
			name:        "custom",
			opts:        []Option{WithLatencyHistogramConfig("db.query.latency", "ms", "query latency", 1, 10, 100)},
			metricName:  "db.query.latency",
			unit:        "ms",
			description: "query latency",
			bounds:      []float64{1, 10, 100},
		},
		{
			name:        "custom without bounds keeps default bounds",
			opts:        []Option{WithLatencyHistogramConfig("db.query.latency", "s", "query latency")},
			metricName:  "db.query.latency",
			unit:        "s",
			description: "query latency",
			bounds:      defaultBucketBoundaries,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			s.query(s.newTracer(tt.opts...), getUserSQL, nil)

			m := s.requireMetric(tt.metricName)
			s.Equal(tt.unit, m.Unit)
			s.Equal(tt.description, m.Description)
			hist, ok := m.Data.(metricdata.Histogram[float64])
			s.Require().True(ok)
			s.Require().Len(hist.DataPoints, 1)
			s.Equal(tt.bounds, hist.DataPoints[0].Bounds)
		})
	}
}

func (s *DBTracerSuite) TestConcurrentQueries() {
	const n = 50

	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.query(s.tracer, getUserSQL, nil)
		}()
	}
	wg.Wait()

	spans := s.spansNamed("postgresql.query")
	s.Len(spans, n)
	for _, span := range spans {
		s.Equal(codes.Ok, span.Status().Code)
		s.assertAttributes(concat(dbAttrs, getUserAttrs, []attribute.KeyValue{
			pgxOperationQuery, semconv.DBOperationName("GetUser"),
		}), attribute.NewSet(span.Attributes()...))
	}

	point := s.requireHistogramPoint(semconv.DBClientOperationDurationName)
	s.Equal(uint64(n), point.Count)
	s.Positive(point.Sum)
	s.assertAttributes(concat(dbAttrs, getUserAttrs, []attribute.KeyValue{
		pgxOperationQuery, PGXStatusKey.String("OK"),
	}), point.Attributes)
	s.Len(s.logs.messages(), n)
}

func (s *DBTracerSuite) query(tracer Tracer, sql string, err error) {
	ctx := tracer.TraceQueryStart(s.ctx, nil, pgx.TraceQueryStartData{SQL: sql, Args: []any{1}})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: err})
}

func (s *DBTracerSuite) spansNamed(name string) []sdktrace.ReadOnlySpan {
	var spans []sdktrace.ReadOnlySpan
	for _, span := range s.spans.Ended() {
		if span.Name() == name {
			spans = append(spans, span)
		}
	}

	return spans
}

func (s *DBTracerSuite) requireSpan(name string) sdktrace.ReadOnlySpan {
	s.T().Helper()

	spans := s.spansNamed(name)
	s.Require().Len(spans, 1, "spans named %q in %v", name, spanNames(s.spans.Ended()))

	return spans[0]
}

func (s *DBTracerSuite) collect() []metricdata.Metrics {
	s.T().Helper()

	var rm metricdata.ResourceMetrics
	s.Require().NoError(s.metrics.Collect(s.ctx, &rm))

	var metrics []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		metrics = append(metrics, sm.Metrics...)
	}

	return metrics
}

func (s *DBTracerSuite) requireMetric(name string) metricdata.Metrics {
	s.T().Helper()

	for _, m := range s.collect() {
		if m.Name == name {
			return m
		}
	}
	s.FailNow("metric not found", name)

	return metricdata.Metrics{}
}

func (s *DBTracerSuite) histogramPoints(name string) []metricdata.HistogramDataPoint[float64] {
	for _, m := range s.collect() {
		if hist, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == name {
			points := hist.DataPoints
			sort.Slice(points, func(i, j int) bool { return statusOf(points[i].Attributes) < statusOf(points[j].Attributes) })
			return points
		}
	}

	return nil
}

func statusOf(attrs attribute.Set) string {
	v, _ := attrs.Value(PGXStatusKey)
	return v.AsString()
}

func (s *DBTracerSuite) requireHistogramPoint(name string) metricdata.HistogramDataPoint[float64] {
	s.T().Helper()

	points := s.histogramPoints(name)
	s.Require().Len(points, 1, "points of %q", name)

	return points[0]
}

func (s *DBTracerSuite) counterPoints(name string) []metricdata.DataPoint[int64] {
	s.T().Helper()

	sum, ok := s.requireMetric(name).Data.(metricdata.Sum[int64])
	s.Require().True(ok, "%q is not an int64 sum", name)
	points := sum.DataPoints
	sort.Slice(points, func(i, j int) bool { return statusOf(points[i].Attributes) < statusOf(points[j].Attributes) })

	return points
}

func (s *DBTracerSuite) requireCounterPoint(name string) metricdata.DataPoint[int64] {
	s.T().Helper()

	points := s.counterPoints(name)
	s.Require().Len(points, 1, "points of %q", name)

	return points[0]
}

func (s *DBTracerSuite) assertAttributes(expected []attribute.KeyValue, actual attribute.Set) {
	s.T().Helper()

	encoder := attribute.DefaultEncoder()
	want := attribute.NewSet(expected...)
	s.Equal(want.Encoded(encoder), actual.Encoded(encoder))
}

type logRecorder struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (h *logRecorder) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *logRecorder) WithGroup(string) slog.Handler            { return h }

func (h *logRecorder) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())

	return nil
}

func (h *logRecorder) messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var messages []string
	for _, r := range h.records {
		messages = append(messages, r.Message)
	}

	return messages
}

func (h *logRecorder) requireOne(t *testing.T) slog.Record {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.records) != 1 {
		t.Fatalf("want 1 log record, got %d", len(h.records))
	}

	return h.records[0]
}

func logAttr(r slog.Record, key string) any {
	var value any
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != key {
			return true
		}
		value = a.Value.Any()
		return false
	})

	return value
}

func batchOf(sqls ...string) *pgx.Batch {
	batch := &pgx.Batch{}
	for _, sql := range sqls {
		batch.Queue(sql)
	}

	return batch
}

func spanNames(spans []sdktrace.ReadOnlySpan) []string {
	names := make([]string, 0, len(spans))
	for _, span := range spans {
		names = append(names, span.Name())
	}

	return names
}

func concat(groups ...[]attribute.KeyValue) []attribute.KeyValue {
	var all []attribute.KeyValue
	for _, g := range groups {
		all = append(all, g...)
	}

	return all
}

func errorsOnly(err error) bool { return err != nil }

func ptr[T any](v T) *T { return &v }
