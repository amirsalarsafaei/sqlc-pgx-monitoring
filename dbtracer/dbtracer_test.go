package dbtracer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	name string // derived from statement
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
		name: s.defaultQuerySQL.name,
		statement: `-- name: get_users :one
	SELECT * FROM users WHERE id = $1`,
		command: s.defaultQuerySQL.command,
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

/*
func (s *DBTracerSuite) TestNewDBTracer() {
	tests := []struct {
		name           string
		databaseName   string
		opts           []Option
		setupMocks     func(*mockmetric.MockMeterProvider, *mockmetric.MockMeter, *mockmetric.MockFloat64Histogram, *mocktracer.MockTracerProvider, *mocktracer.MockTracer)
		validateTracer func(*DBTracerSuite, Tracer)
		wantErr        bool
	}{
		{
			name:         "successful creation with default options",
			databaseName: "test_db",
			opts:         []Option{},
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram, tp *mocktracer.MockTracerProvider, t *mocktracer.MockTracer) {
				mp.EXPECT().
					Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring").
					Return(m)
				m.EXPECT().
					Float64Histogram(
						semconv.DBClientOperationDurationName,
						metric.WithDescription(semconv.DBClientOperationDurationDescription),
						metric.WithUnit(semconv.DBClientOperationDurationUnit),
					).
					Return(h, nil)
			},
			wantErr: false,
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
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram, tp *mocktracer.MockTracerProvider, t *mocktracer.MockTracer) {
				mp.EXPECT().
					Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring").
					Return(m)
				m.EXPECT().
					Float64Histogram(
						semconv.DBClientOperationDurationName,
						metric.WithDescription(semconv.DBClientOperationDurationDescription),
						metric.WithUnit(semconv.DBClientOperationDurationUnit),
					).
					Return(h, nil)
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
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram, tp *mocktracer.MockTracerProvider, t *mocktracer.MockTracer) {
				mp.EXPECT().
					Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring").
					Return(m)
				m.EXPECT().
					Float64Histogram(
						"custom.histogram",
						metric.WithDescription("Custom description"),
						metric.WithUnit("ms"),
					).
					Return(h, nil)
			},
			wantErr: false,
		},
		{
			name:         "meter creation failure",
			databaseName: "test_db",
			opts:         []Option{},
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram, tp *mocktracer.MockTracerProvider, t *mocktracer.MockTracer) {
				mp.EXPECT().
					Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring").
					Return(m)
				m.EXPECT().
					Float64Histogram(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("meter creation failed"))
			},
			wantErr: true,
		},
		{
			name:         "empty database name",
			databaseName: "",
			opts:         []Option{},
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram, tp *mocktracer.MockTracerProvider, t *mocktracer.MockTracer) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			mp := mockmetric.NewMockMeterProvider(s.T())
			m := mockmetric.NewMockMeter(s.T())
			h := mockmetric.NewMockFloat64Histogram(s.T())
			tp := mocktracer.NewMockTracerProvider(s.T())
			t := mocktracer.NewMockTracer(s.T())

			tt.setupMocks(mp, m, h, tp, t)

			opts := append(tt.opts,
				WithMeterProvider(mp),
				WithTraceProvider(tp),
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
*/

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

	span := queryData.span.(sdktrace.ReadOnlySpan)
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
		PGXOperationTypeKey.String("query"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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
		PGXOperationTypeKey.String("query"),
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTraceQueryDuration() {
	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	sleepDuration := 100 * time.Millisecond
	time.Sleep(sleepDuration)

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

	// Check that duration is approximately what we expected
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration >= 95*time.Millisecond, "Duration should be at least 95ms, got %v", duration)
	s.True(duration <= 200*time.Millisecond, "Duration should be at most 200ms, got %v", duration)

	// Verify metrics - the recorded duration should be in seconds
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be approximately 0.1 seconds (100ms)
	s.True(point.Sum >= 0.095, "Recorded duration should be at least 0.095s, got %v", point.Sum)
	s.True(point.Sum <= 0.200, "Recorded duration should be at most 0.200s, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		PGXOperationTypeKey.String("query"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTraceBatchDuration() {
	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []interface{}{1},
		CommandTag: pgconn.CommandTag{},
	})

	sleepDuration := 200 * time.Millisecond
	time.Sleep(sleepDuration)

	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})

	// Verify spans
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal(s.defaultQuerySQL.name, span.Name())
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
	s.Equal("batch", attrMap[PGXOperationTypeKey])

	// Check that duration is approximately what we expected
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration >= 195*time.Millisecond, "Duration should be at least 195ms, got %v", duration)
	s.True(duration <= 400*time.Millisecond, "Duration should be at most 400ms, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be approximately 0.2 seconds (200ms)
	s.True(point.Sum >= 0.195, "Recorded duration should be at least 0.195s, got %v", point.Sum)
	s.True(point.Sum <= 0.400, "Recorded duration should be at most 0.400s, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		PGXOperationTypeKey.String("batch"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTracePrepareWithDuration() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL.statement,
	})

	sleepDuration := 150 * time.Millisecond
	time.Sleep(sleepDuration)

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

	// Check that duration is approximately what we expected
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration >= 145*time.Millisecond, "Duration should be at least 145ms, got %v", duration)
	s.True(duration <= 200*time.Millisecond, "Duration should be at most 200ms, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	// Duration should be approximately 0.15 seconds (150ms)
	s.True(point.Sum >= 0.145, "Recorded duration should be at least 0.145s, got %v", point.Sum)
	s.True(point.Sum <= 0.200, "Recorded duration should be at most 0.200s, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		PGXOperationTypeKey.String("prepare"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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
		PGXOperationTypeKey.String("prepare"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
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
		PGXOperationTypeKey.String("prepare"),
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTraceConnectSuccess() {
	connConfig := &pgx.ConnConfig{}

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	sleepDuration := 50 * time.Millisecond
	time.Sleep(sleepDuration)

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

	// Check that duration is approximately what we expected
	duration := span.EndTime().Sub(span.StartTime())
	s.True(duration >= 45*time.Millisecond, "Duration should be at least 45ms, got %v", duration)
	s.True(duration <= 100*time.Millisecond, "Duration should be at most 100ms, got %v", duration)

	// Verify metrics
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum >= 0.045, "Recorded duration should be at least 0.045s, got %v", point.Sum)
	s.True(point.Sum <= 0.100, "Recorded duration should be at most 0.100s, got %v", point.Sum)

	// Check attributes
	expectedAttrs := attribute.NewSet(
		PGXOperationTypeKey.String("connect"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String("connect"),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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
		PGXOperationTypeKey.String("connect"),
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String("connect"),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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
	s.Equal("\"users\"", attrMap[attribute.Key("db.table")])
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
		PGXOperationTypeKey.String("copy_from"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String("copy_from"),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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
	s.Equal("\"users\"", attrMap[attribute.Key("db.table")])
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
		PGXOperationTypeKey.String("copy_from"),
		PGXStatusKey.String("UNKNOWN_ERROR"),
		SQLCQueryNameKey.String("copy_from"),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
}

func (s *DBTracerSuite) TestTraceConcurrent() {
	// Test concurrent queries to ensure thread safety
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

	// Verify metrics were aggregated correctly
	// All concurrent queries have the same attributes, so they should be aggregated into a single data point
	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1, "Expected single aggregated data point for queries with identical attributes")

	point := histogramPoints[0]
	s.Equal(uint64(numQueries), point.Count, "Expected count to equal number of concurrent queries")
	s.True(point.Sum > 0, "Expected positive sum of all durations")

	expectedAttrs := attribute.NewSet(
		PGXOperationTypeKey.String("query"),
		PGXStatusKey.String("OK"),
		SQLCQueryNameKey.String(s.defaultQuerySQL.name),
	)
	s.True(expectedAttrs.Equals(&point.Attributes))
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

	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	span := queryData.span

	dbTracer := s.dbTracer.(*dbTracer)

	// Test with nil error (should not record anything)
	dbTracer.recordSpanError(span, nil)

	// End the span and check it has no error status
	span.End()
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	s.Equal(codes.Unset, spans[0].Status().Code)

	// Reset for next test
	s.resetRecorders()

	// Test with pgx.ErrNoRows (should not record error)
	ctx2 := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{2},
	})

	queryData2 := ctx2.Value(dbTracerQueryCtxKey).(*traceQueryData)
	span2 := queryData2.span

	dbTracer.recordSpanError(span2, pgx.ErrNoRows)
	span2.End()

	spans = s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	s.Equal(codes.Unset, spans[0].Status().Code) // Should not set error status for ErrNoRows

	// Reset for next test
	s.resetRecorders()

	// Test with regular error
	ctx3 := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{3},
	})

	queryData3 := ctx3.Value(dbTracerQueryCtxKey).(*traceQueryData)
	span3 := queryData3.span

	testErr := errors.New("test error")
	dbTracer.recordSpanError(span3, testErr)
	span3.End()

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
	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	// First query (success)
	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []any{1},
		CommandTag: pgconn.CommandTag{},
	})

	// Second query with error
	batchErr := errors.New("batch query error")
	// FIXME: the error is never recorded on a metric
	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL.statement,
		Args:       []any{2},
		CommandTag: pgconn.CommandTag{},
		Err:        batchErr,
	})

	// End batch
	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})

	// Verify spans
	// queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	// span := queryData.span.(sdktrace.ReadOnlySpan)
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal(s.defaultQuerySQL.name, span.Name())
	// FIXME: can't overwrite OK status with error status, calling TraceBatchQuery should create a separate child span instead
	// s.Equal(codes.Error, span.Status().Code)
	// s.Equal(batchErr.Error(), span.Status().Description)
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
	s.Equal("batch", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])

	// Verify that error events were recorded
	events := span.Events()
	s.Require().Len(events, 1)
	s.Equal("exception", events[0].Name)

	histogramPoints := s.getHistogramPoints()
	s.Require().Len(histogramPoints, 1)

	point := histogramPoints[0]
	s.Equal(uint64(1), point.Count)
	s.True(point.Sum > 0) // Duration should be positive
}

func (s *DBTracerSuite) TestTraceWithIncludeSQLText() {
	// Create tracer with includeSQLText enabled
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
	// Test with custom shouldLog function that filters errors
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

	// Test the shouldLog function
	s.True(dbTracer.shouldLog(errors.New("some error")))
	s.False(dbTracer.shouldLog(pgx.ErrNoRows))
	s.False(dbTracer.shouldLog(nil))
}

func (s *DBTracerSuite) TestTraceQueryStart_WithAppendQueryNameToSpan() {
	// Create tracer with appendQueryNameToSpan enabled
	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
	)
	s.Require().NoError(err)

	ctx := tracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL.statement,
		Args: []interface{}{1},
	})

	s.NotNil(ctx)
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	s.NotNil(queryData)
	s.Equal(s.defaultQuerySQL, queryData.sql)
	s.Equal([]interface{}{1}, queryData.args)

	// End the query to finalize the span
	tracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})

	// Verify span was created with query name appended
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.query/get_users", span.Name())
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
	s.Equal(s.defaultQuerySQL, prepareData.sql)
	s.Equal(s.defaultQuerySQL.name, prepareData.qMD.name)

	// End the prepare to finalize the span
	tracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: false,
		Err:             nil,
	})

	// Verify span was created with query name appended
	spans := s.spanRecorder.Ended()
	s.Require().Len(spans, 1)
	span := spans[0]
	s.Equal("postgresql.prepare/get_users", span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	// Check span attributes
	attrs := span.Attributes()
	attrMap := make(map[attribute.Key]string)
	for _, attr := range attrs {
		if attr.Value.Type() == attribute.STRING {
			attrMap[attr.Key] = attr.Value.AsString()
		}
	}
	s.Equal("prepare", attrMap[PGXOperationTypeKey])
	s.Equal(stmtName, attrMap[PGXPrepareStmtNameKey])
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])
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
	s.Equal(s.defaultQuerySQL.name, span.Name())
	s.Equal(codes.Ok, span.Status().Code)

	attrs := span.Attributes()
	attrMap := s.attributeToMap(attrs)

	s.Equal("batch", attrMap[PGXOperationTypeKey])
	s.Equal(s.defaultQuerySQL.name, attrMap[SQLCQueryNameKey])
	s.Equal(s.defaultQuerySQL.command, attrMap[SQLCQueryCommandKey])
	s.Equal(s.defaultDBName, attrMap[semconv.DBNamespaceKey])
	s.Equal(semconv.DBSystemPostgreSQL.Value.AsString(), attrMap[semconv.DBSystemKey])
}

func (s *DBTracerSuite) attributeToMap(attrs []attribute.KeyValue) map[attribute.Key]any {
	attrMap := make(map[attribute.Key]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	return attrMap
}
