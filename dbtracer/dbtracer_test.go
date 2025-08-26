package dbtracer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type queryTestData struct {
	statement string
	command   string // derived from statement
	name      string // derived from statement
}

type DBTracerSuite struct {
	suite.Suite
	tracerProvider  trace.TracerProvider
	spanRecorder    *tracetest.SpanRecorder
	meter           metric.Meter
	meterProvider   metric.MeterProvider
	metricReader    *sdkmetric.ManualReader
	histogram       metric.Float64Histogram
	shouldLog       func() ShouldLog
	logger          *slog.Logger
	ctx             context.Context
	pgxConn         *pgx.Conn
	pgxPool         *pgxpool.Pool
	dbTracer        Tracer
	defaultDBName   string
	defaultQuerySQL queryTestData
}

// getMetrics retrieves metrics from the manual reader for inspection
func (s *DBTracerSuite) getMetrics() []metricdata.Metrics {
	var rm metricdata.ResourceMetrics
	err := s.metricReader.Collect(context.Background(), &rm)
	s.Require().NoError(err)

	var metrics []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		metrics = append(metrics, sm.Metrics...)
	}
	return metrics
}

// getHistogramPoints retrieves histogram data points from metrics
func (s *DBTracerSuite) getHistogramPoints() []metricdata.HistogramDataPoint[float64] {
	metrics := s.getMetrics()
	for _, m := range metrics {
		if h, ok := m.Data.(metricdata.Histogram[float64]); ok {
			return h.DataPoints
		}
	}
	return nil
}

func TestDBTracerSuite(t *testing.T) {
	suite.Run(t, new(DBTracerSuite))
}

// resetRecorders clears the span recorder and metric reader for a clean test state
func (s *DBTracerSuite) resetRecorders() {
	s.spanRecorder.Reset()

	// Collect metrics to clear them
	var rm metricdata.ResourceMetrics
	_ = s.metricReader.Collect(context.Background(), &rm)
}

func (s *DBTracerSuite) SetupTest() {
	s.T().Helper()

	s.ctx = context.Background()
	s.defaultDBName = "test_db"
	s.defaultQuerySQL = queryTestData{
		name: "get_users",
		statement: `-- name: get_users :one
	SELECT * FROM users WHERE id = $1`,
		command: "one",
	}

	s.spanRecorder = tracetest.NewSpanRecorder()
	s.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(s.spanRecorder),
	)

	// Initialize metric provider with manual reader for testing
	s.metricReader = sdkmetric.NewManualReader()
	s.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(s.metricReader),
	)
	s.meter = s.meterProvider.Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring")

	s.pgxConn = &pgx.Conn{}
	s.pgxPool = &pgxpool.Pool{}

	var err error
	s.histogram, err = s.meter.Float64Histogram(
		semconv.DBClientOperationDurationName,
		metric.WithDescription(semconv.DBClientOperationDurationDescription),
		metric.WithUnit(semconv.DBClientOperationDurationUnit),
	)
	s.Require().NoError(err)

	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	s.shouldLog = func() ShouldLog {
		return func(err error) bool {
			return true
		}
	}

	s.dbTracer, err = NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithLogger(s.logger),
		WithShouldLog(s.shouldLog()),
	)
	s.Require().NoError(err)

	// Start with clean recorders
	s.resetRecorders()
}

func (s *DBTracerSuite) TestNewDBTracer() {
	tests := []struct {
		name           string
		databaseName   string
		opts           []Option
		validateTracer func(*DBTracerSuite, Tracer)
		wantErr        bool
	}{
		{
			name:         "successful creation with default options",
			databaseName: "test_db",
			opts:         []Option{},
			wantErr:      false,
			validateTracer: func(s *DBTracerSuite, t Tracer) {
				dbTracer, ok := t.(*dbTracer)
				s.Require().True(ok)
				s.Equal(
					slog.Default(),
					dbTracer.logger,
					"Should use default logger when none specified",
				)
			},
		},
		{
			name:         "successful creation with custom logger",
			databaseName: "test_db",
			opts: []Option{
				WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
			},
			validateTracer: func(s *DBTracerSuite, t Tracer) {
				dbTracer, ok := t.(*dbTracer)
				s.Require().True(ok)
				s.NotEqual(slog.Default(), dbTracer.logger, "Should use custom logger")
			},
			wantErr: false,
		},
		{
			name:         "successful creation with custom histogram config",
			databaseName: "test_db",
			opts: []Option{
				WithLatencyHistogramConfig("custom.histogram", "ms", "Custom description"),
			},
			wantErr: false,
		},
		{
			name:         "successful creation with all options",
			databaseName: "test_db",
			opts: []Option{
				WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
				WithShouldLog(func(err error) bool { return err != nil }),
				WithLogArgs(true),
				WithLogArgsLenLimit(256),
				WithIncludeSQLText(true),
				WithLatencyHistogramConfig("custom.duration", "s", "Custom duration metric",
					0.1, 10, 100, 1000),
			},
			wantErr: false,
			validateTracer: func(s *DBTracerSuite, t Tracer) {
				dbTracer, ok := t.(*dbTracer)
				s.Require().True(ok)
				s.NotEqual(slog.Default(), dbTracer.logger, "Should use custom logger")
				s.True(dbTracer.logArgs, "Should enable log args")
				s.Equal(256, dbTracer.logArgsLenLimit, "Should set custom args length limit")
				s.True(dbTracer.includeQueryText, "Should include SQL text")
			},
		},
		{
			name:         "empty database name",
			databaseName: "",
			opts:         []Option{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Use real providers from the suite setup
			opts := append(tt.opts,
				WithMeterProvider(s.meterProvider),
				WithTraceProvider(s.tracerProvider),
			)
			tracer, err := NewDBTracer(tt.databaseName, opts...)

			if tt.wantErr {
				s.Error(err)
				s.Nil(tracer)
			} else {
				s.NoError(err)
				s.NotNil(tracer)
				if tt.validateTracer != nil {
					tt.validateTracer(s, tracer)
				}
			}
		})
	}
}

func (s *DBTracerSuite) TestTraceQueryStart() {
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	s.NotNil(ctx)
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	s.NotNil(queryData)
	s.Equal(s.defaultQuerySQL.statement, queryData.sql)
	s.Equal([]interface{}{1}, queryData.args)

	// Verify that a span was started (but not ended yet)
	spans := s.spanRecorder.Ended()
	s.Len(spans, 0, "span should not be ended yet")

	span := trace.SpanFromContext(ctx).(sdktrace.ReadOnlySpan)
	s.Equal("postgresql.query", span.Name())

	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("query", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) TestTraceQueryEnd_Success() {
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.query", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("query", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationQuery,
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceQueryEnd_Error() {
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	expectedErr := errors.New("database error")

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.query", span.Name())
	s.Equal(codes.Error, span.Status().Code)
	s.Equal(expectedErr.Error(), span.Status().Description)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("query", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify that the span has recorded events (error recording)
	events := span.Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationQuery,
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceQueryDuration() {
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.query", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check that duration is positive (actual execution time)
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration > 0, "Duration should be positive, got %v", duration)

	// Verify metrics - the recorded duration should be in seconds
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be positive
	s.True(point.Sum > 0, "Recorded duration should be positive, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationQuery,
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceBatchDuration() {
	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []interface{}{1},
		CommandTag: pgconn.CommandTag{},
	})

	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.batch", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal("batch", attrMap[PGXOperationTypeKey])

	// Check that duration is positive (actual execution time)
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration > 0, "Duration should be positive, got %v", duration)

	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be positive
	s.True(point.Sum > 0, "Recorded duration should be positive, got %v", point.Sum)

	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationBatch,
		PGXStatusKey.String("OK"),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTracePrepareWithDuration() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL.statement,
	})

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: false,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.prepare", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("prepare", attrMap[PGXOperationTypeKey])
	s.Equal(stmtName, attrMap[PGXPrepareStmtNameKey])

	// Check that duration is positive (actual execution time)
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration > 0, "Duration should be positive, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be positive
	s.True(point.Sum > 0, "Recorded duration should be positive, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationPrepare,
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTracePrepareAlreadyPrepared() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL.statement,
	})

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: true,
	})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.prepare", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("prepare", attrMap[PGXOperationTypeKey])
	s.Equal(stmtName, attrMap[PGXPrepareStmtNameKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationPrepare,
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTracePrepareError() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"
	expectedErr := errors.New("prepare failed")

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL.statement,
	})

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		Err: expectedErr,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.prepare", span.Name())
	s.Equal(codes.Error, span.Status().Code)
	s.Equal(expectedErr.Error(), span.Status().Description)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("prepare", attrMap[PGXOperationTypeKey])
	s.Equal(stmtName, attrMap[PGXPrepareStmtNameKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify that the span has recorded events (error recording)
	events := span.Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationPrepare,
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceConnectSuccess() {
	connConfig := &pgx.ConnConfig{}

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	s.dbTracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{
		Conn: s.pgxConn,
		Err:  nil,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.connect", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Check that duration is positive (actual execution time)
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration > 0, "Duration should be positive, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0, "Recorded duration should be positive, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationConnect,
		PGXStatusKey.String("OK"),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceConnectError() {
	connConfig := &pgx.ConnConfig{}
	expectedErr := errors.New("connection failed")

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	s.dbTracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{
		Conn: nil,
		Err:  expectedErr,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.connect", span.Name())
	s.Equal(codes.Error, span.Status().Code)
	s.Equal(expectedErr.Error(), span.Status().Description)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify that the span has recorded events (error recording)
	events := span.Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationConnect,
		PGXStatusKey.String("UNKNOWN_ERROR"),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceCopyFromSuccess() {
	tableName := pgx.Identifier{"users"}
	columnNames := []string{"id", "name"}

	ctx := s.dbTracer.TraceCopyFromStart(s.ctx, s.pgxConn, pgx.TraceCopyFromStartData{
		TableName:   tableName,
		ColumnNames: columnNames,
	})

	s.dbTracer.TraceCopyFromEnd(ctx, s.pgxConn, pgx.TraceCopyFromEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.copy_from", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal("copy_from", attrMap[PGXOperationTypeKey])
	s.Equal("\"users\"", attrMap[semconv.DBCollectionNameKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Check that duration is positive
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration > 0, "Duration should be positive, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0, "Recorded duration should be positive, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationCopyFrom,
		PGXStatusKey.String("OK"),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceCopyFromError() {
	tableName := pgx.Identifier{"users"}
	columnNames := []string{"id", "name"}
	expectedErr := errors.New("copy failed")

	ctx := s.dbTracer.TraceCopyFromStart(s.ctx, s.pgxConn, pgx.TraceCopyFromStartData{
		TableName:   tableName,
		ColumnNames: columnNames,
	})

	s.dbTracer.TraceCopyFromEnd(ctx, s.pgxConn, pgx.TraceCopyFromEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.copy_from", span.Name())
	s.Equal(codes.Error, span.Status().Code)
	s.Equal(expectedErr.Error(), span.Status().Description)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal("copy_from", attrMap[PGXOperationTypeKey])
	s.Equal("\"users\"", attrMap[semconv.DBCollectionNameKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify that the span has recorded events (error recording)
	events := span.Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive

	// Check attributes
	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationCopyFrom,
		PGXStatusKey.String("UNKNOWN_ERROR"),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestTraceConcurrent() {
	const numQueries = 10
	var wg sync.WaitGroup

	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func(queryID int) {
			defer wg.Done()

			ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
				SQL:  s.defaultQuerySQL.statement,
				Args: []interface{}{queryID},
			})

			s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
				CommandTag: pgconn.CommandTag{},
				Err:        nil,
			})
		}(i)
	}

	wg.Wait()

	// Verify all spans were created
	spans := s.spanRecorder.Ended()
	s.Len(spans, numQueries)

	// Verify all spans have correct attributes
	for _, span := range spans {
		s.Equal("postgresql.query", span.Name())
		s.Equal(codes.Ok, span.Status().Code)

		attrs := span.Attributes()
		attrMap := s.attributesToMap(attrs)
		s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey].AsString())
		s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey].AsString())
		s.Equal("query", attrMap[PGXOperationTypeKey].AsString())
		s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
		s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey].AsString())
	}

	// Verify metrics were aggregated correctly
	// All concurrent queries have the same attributes, so they should be aggregated into a single data point
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1, "Expected single aggregated data point for queries with identical attributes")

	point := histogramPoints[0]
	s.Equal(uint64(numQueries), point.Count, "Expected count to equal number of concurrent queries")
	s.True(point.Sum > 0, "Expected positive sum of all durations")

	expectedAttrs := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(s.defaultDBName),
		pgxOperationQuery,
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
		SQLCQueryCommandKey.String(s.defaultQuerySQL.command),
	)
	s.EqualAttributeSet(expectedAttrs, point.Attributes)
}

func (s *DBTracerSuite) TestLoggerBehavior() {
	// Test that logger can be accessed through the tracer
	dbTracer, ok := s.dbTracer.(*dbTracer)
	s.Require().True(ok)
	s.NotNil(dbTracer.logger)
}

// Test coverage for missing options and edge cases
func (s *DBTracerSuite) TestNewDBTracerWithAllOptions() {
	customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	customShouldLog := func(err error) bool { return err != nil }

	tracer, err := NewDBTracer(
		"test_db",
		WithLogger(customLogger),
		WithShouldLog(customShouldLog),
		WithMeterProvider(s.meterProvider),
		WithTraceProvider(s.tracerProvider),
		WithLogArgs(false),
		WithLogArgsLenLimit(128),
		WithIncludeSQLText(true),
		WithLatencyHistogramConfig("custom.duration", "ms", "Custom duration metric"),
	)

	s.NoError(err)
	s.NotNil(tracer)

	dbTracer, ok := tracer.(*dbTracer)
	s.True(ok)
	s.Equal(customLogger, dbTracer.logger)
	s.False(dbTracer.logArgs)
	s.Equal(128, dbTracer.logArgsLenLimit)
	s.True(dbTracer.includeQueryText)
}

func (s *DBTracerSuite) TestLogQueryArgsWithDifferentTypes() {
	// Test with logArgs disabled
	tracer, err := NewDBTracer(
		"test_db",
		WithLogArgs(false),
		WithMeterProvider(s.meterProvider),
		WithTraceProvider(s.tracerProvider),
	)
	s.NoError(err)

	dbTracerImpl := tracer.(*dbTracer)

	// Should return nil when logArgs is false
	result := dbTracerImpl.logQueryArgs([]any{"test", 123, []byte("data")})
	s.Nil(result)

	// Test with logArgs enabled and different data types
	tracer2, err := NewDBTracer(
		"test_db",
		WithLogArgs(true),
		WithLogArgsLenLimit(10),
		WithMeterProvider(s.meterProvider),
		WithTraceProvider(s.tracerProvider),
	)
	s.NoError(err)

	dbTracer2 := tracer2.(*dbTracer)

	// Test with various data types
	args := []any{
		"short",
		"this is a very long string that should be truncated",
		[]byte("short"),
		[]byte("this is a very long byte array that should be truncated"),
		123,
		nil,
	}

	result = dbTracer2.logQueryArgs(args)
	s.NotNil(result)
	s.Len(result, len(args))

	// Check that long string was truncated
	s.Contains(result[1].(string), "truncated")
	// Check that long byte array was truncated
	s.Contains(result[3].(string), "truncated")
}

func (s *DBTracerSuite) TestExtractConnectionID() {
	// Test with nil connection
	id := extractConnectionID(nil)
	s.Equal(uint32(0), id)

	// Note: Testing with real pgx.Conn is complex due to its internal structure
	// The function is designed to handle nil gracefully, which we've tested
}

func (s *DBTracerSuite) TestPgxStatusFromErr() {
	// Test with nil error
	status := pgxStatusFromErr(nil)
	s.Equal("OK", status)

	// Test with generic error
	genericErr := errors.New("generic error")
	status = pgxStatusFromErr(genericErr)
	s.Equal("UNKNOWN_ERROR", status)

	// Test with pgx.ErrNoRows (should still return UNKNOWN_ERROR)
	status = pgxStatusFromErr(pgx.ErrNoRows)
	s.Equal("UNKNOWN_ERROR", status)
}

func (s *DBTracerSuite) TestRecordSpanErrorWithPgError() {
	// Start a query to get a real span
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	span := trace.SpanFromContext(ctx)

	dbTracer := s.dbTracer.(*dbTracer)

	// Test with nil error (should not record anything)
	dbTracer.recordSpanError(span, nil)

	// End the span and check it has no error status
	span.End()
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	s.Equal(codes.Unset, spans[0].Status().Code)

	s.resetRecorders()

	ctx = s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{2},
	})

	span = trace.SpanFromContext(ctx)

	span.End()

	spans = s.spanRecorder.Ended()
	s.Require().Len(spans, 1)

	s.resetRecorders()

	ctx = s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []any{3},
	})

	span = trace.SpanFromContext(ctx)

	testErr := errors.New("test error")
	dbTracer.recordSpanError(span, testErr)
	span.End()

	spans = s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	s.Equal(codes.Error, spans[0].Status().Code)
	s.Equal("test error", spans[0].Status().Description)

	// Verify that the error was recorded as an event
	events := spans[0].Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)
}

func (s *DBTracerSuite) TestTraceBatchWithMultipleQueries() {
	// Start batch
	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{
		Batch: &pgx.Batch{
			QueuedQueries: []*pgx.QueuedQuery{
				{
					SQL:       s.defaultQuerySQL.statement,
					Arguments: []any{1},
					Fn:        nil,
				},
				{
					SQL:       s.defaultQuerySQL.statement,
					Arguments: []any{2},
					Fn:        nil,
				},
			},
		},
	})

	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []any{1},
		CommandTag: pgconn.CommandTag{},
	})

	batchErr := errors.New("batch query error")
	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []any{2},
		CommandTag: pgconn.CommandTag{},
		Err:        batchErr,
	})

	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{Err: batchErr})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 3)

	s.Equal("postgresql.batch.query", spans[0].Name())
	s.Equal(codes.Ok, spans[0].Status().Code)

	s.Equal("postgresql.batch.query", spans[1].Name())
	s.Equal(codes.Error, spans[1].Status().Code)

	s.Equal("postgresql.batch", spans[2].Name())
	s.Equal(codes.Error, spans[2].Status().Code)

	// check one of batch queries attrs
	attrs := spans[0].Attributes()
	attrMap := s.attributesToMap(attrs)
	s.Equal("batch.query", attrMap[PGXOperationTypeKey].AsString())
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
	s.Equal(semconv.DBSystemPostgreSQL.Value, attrMap[semconv.DBSystemKey])
	s.Equal(s.defaultQuerySQL.name, attrMap[semconv.DBOperationNameKey].AsString())
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey].AsString())
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey].AsString())

	// check batch end attrs
	attrs = spans[2].Attributes()
	attrMap = s.attributesToMap(attrs)

	s.Equal("batch", attrMap[PGXOperationTypeKey].AsString())
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
	s.Equal(semconv.DBSystemPostgreSQL.Value, attrMap[semconv.DBSystemKey])

	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive
}

func (s *DBTracerSuite) TestTraceWithIncludeSQLText() {
	tracer, err := NewDBTracer(
		"test_db",
		WithIncludeSQLText(true),
		WithMeterProvider(s.meterProvider),
		WithTraceProvider(s.tracerProvider),
	)
	s.NoError(err)

	dbTracer := tracer.(*dbTracer)
	s.True(dbTracer.includeQueryText)
}

func (s *DBTracerSuite) TestShouldLogFunctionality() {
	customShouldLog := func(err error) bool {
		return err != nil && !errors.Is(err, pgx.ErrNoRows)
	}

	tracer, err := NewDBTracer(
		"test_db",
		WithShouldLog(customShouldLog),
		WithMeterProvider(s.meterProvider),
		WithTraceProvider(s.tracerProvider),
	)
	s.NoError(err)

	dbTracer := tracer.(*dbTracer)

	s.True(dbTracer.shouldLog(errors.New("some error")))
	s.False(dbTracer.shouldLog(pgx.ErrNoRows))
	s.False(dbTracer.shouldLog(nil))
}

func (s *DBTracerSuite) TestTraceQueryStart_WithSpanNameSuffix() {
	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
		WithIncludeSpanNameSuffix(true),
	)
	s.Require().NoError(err)

	ctx := tracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	s.NotNil(ctx)
	traceData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	s.NotNil(traceData)
	s.Equal(s.defaultQuerySQL.statement, traceData.sql)
	s.Equal([]interface{}{1}, traceData.args)

	tracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.query get_users", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal("query", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) TestTracePrepareStart() {
	// Create tracer with appendQueryNameToSpan enabled
	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
	)
	s.Require().NoError(err)

	stmtName := "get_user_by_id"

	ctx := tracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  s.defaultQuerySQL.statement,
	})

	s.NotNil(ctx)
	prepareData := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)
	s.NotNil(prepareData)
	s.Equal(s.defaultQuerySQL.statement, prepareData.sql)
	s.Equal(s.defaultQuerySQL.name, prepareData.qMD.name)

	tracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: false,
		Err:             nil,
	})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.prepare", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	attrMap := s.attributesToMap(attrs)

	s.Equal("prepare", attrMap[PGXOperationTypeKey].AsString())
	s.Equal(stmtName, attrMap[PGXPrepareStmtNameKey].AsString())
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey].AsString())
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey].AsString())
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
	s.Equal(semconv.DBSystemPostgreSQL.Value, attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) TestTraceBatchQuery() {
	// Create tracer with appendQueryNameToSpan enabled
	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
	)
	s.Require().NoError(err)

	ctx := tracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	tracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []any{1},
		CommandTag: pgconn.CommandTag{},
	})

	tracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.batch", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	attrMap := s.attributesToMap(attrs)

	s.Equal("batch", attrMap[PGXOperationTypeKey].AsString())
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
	s.Equal(semconv.DBSystemPostgreSQL.Value, attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) TestTraceAcquire() {
	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
	)
	s.Require().NoError(err)

	ctx := tracer.TraceAcquireStart(s.ctx, s.pgxPool, pgxpool.TraceAcquireStartData{})
	tracer.TraceAcquireEnd(ctx, s.pgxPool, pgxpool.TraceAcquireEndData{Conn: s.pgxConn, Err: nil})

	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)

	span := spans[0]
	s.Equal("pgxpool.acquire", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	s.Len(attrs, 3)

	attrMap := s.attributesToMap(attrs)

	s.Equal("acquire", attrMap[PGXPoolConnOperationKey].AsString())
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey].AsString())
	s.Equal(semconv.DBSystemPostgreSQL.Value, attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) attributesToMap(attrs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	attrMap := make(map[attribute.Key]attribute.Value)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	return attrMap
}

func (s *DBTracerSuite) EqualAttributeSet(expected, actual attribute.Set, msgAndArgs ...any) bool {
	s.T().Helper()

	if !expected.Equals(&actual) {
		encoder := attribute.DefaultEncoder()

		expectedStr := expected.Encoded(encoder)
		actualStr := actual.Encoded(encoder)

		return assert.Fail(s.T(), fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s%s", expectedStr, actualStr, diffString(expectedStr, actualStr)),
			msgAndArgs...)
	}

	return true
}

func diffString(expected, actual string) string {

	var e, a string

	e = reflect.ValueOf(expected).String()
	a = reflect.ValueOf(actual).String()

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(e),
		B:        difflib.SplitLines(a),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return "\n\nDiff:\n" + diff
}
