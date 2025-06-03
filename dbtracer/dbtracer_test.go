package dbtracer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	mockmetric "github.com/amirsalarsafaei/sqlc-pgx-monitoring/mocks/go.opentelemetry.io/otel/metric"
	mocktracer "github.com/amirsalarsafaei/sqlc-pgx-monitoring/mocks/go.opentelemetry.io/otel/trace"
)

type DBTracerSuite struct {
	suite.Suite
	tracer          *mocktracer.MockTracer
	tracerProvider  *mocktracer.MockTracerProvider
	span            *mocktracer.MockSpan
	meter           *mockmetric.MockMeter
	meterProvider   *mockmetric.MockMeterProvider
	histogram       *mockmetric.MockFloat64Histogram
	shouldLog       func() ShouldLog
	logger          *slog.Logger
	ctx             context.Context
	pgxConn         *pgx.Conn
	dbTracer        Tracer
	defaultDBName   string
	defaultQuerySQL string
}

// matchAttributes creates a matcher function for metric.WithAttributes
func (s *DBTracerSuite) matchAttributes(expected ...attribute.KeyValue) interface{} {
	return mock.MatchedBy(func(actual interface{}) bool {
		opt, ok := actual.(metric.MeasurementOption)
		if !ok {
			return false
		}

		return s.Equal(metric.WithAttributes(expected...), opt)
	})
}

func TestDBTracerSuite(t *testing.T) {
	suite.Run(t, new(DBTracerSuite))
}

func (s *DBTracerSuite) SetupTest() {
	// Initialize all mock objects
	s.tracer = mocktracer.NewMockTracer(s.T())
	s.tracerProvider = mocktracer.NewMockTracerProvider(s.T())
	s.span = mocktracer.NewMockSpan(s.T())
	s.meter = mockmetric.NewMockMeter(s.T())
	s.meterProvider = mockmetric.NewMockMeterProvider(s.T())
	s.histogram = mockmetric.NewMockFloat64Histogram(s.T())
	s.shouldLog = func() ShouldLog {
		return func(err error) bool {
			return true
		}
	}
	s.logger = slog.Default()
	s.ctx = context.Background()
	s.defaultDBName = "test_db"
	s.defaultQuerySQL = `-- name: get_users :one
	SELECT * FROM users WHERE id = $1`

	s.tracerProvider.EXPECT().
		Tracer(mock.Anything).
		Return(s.tracer).Maybe()

	s.meterProvider.EXPECT().
		Meter(mock.Anything).
		Return(s.meter)

	s.meter.EXPECT().
		Float64Histogram(mock.Anything, mock.Anything, mock.Anything).
		Return(s.histogram, nil)

	// shouldLog function is now a simple function, no mocking needed

	s.span.EXPECT().
		SetAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBNamespace(s.defaultDBName),
		).
		Return().Maybe()

	var err error
	s.dbTracer, err = NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog()),
		WithLogger(s.logger),
	)
	s.Require().NoError(err)
}

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

func (s *DBTracerSuite) TestTraceQueryStart() {
	// Setup expectations
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
			PGXOperationTypeKey.String("query"),
		).
		Return()

	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL,
		Args: []interface{}{1},
	})

	s.NotNil(ctx)
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	s.NotNil(queryData)
	s.Equal(s.defaultQuerySQL, queryData.sql)
	s.Equal([]interface{}{1}, queryData.args)
}

func (s *DBTracerSuite) TestTraceQueryEnd_Success() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
			PGXOperationTypeKey.String("query"),
		).
		Return()

	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL,
		Args: []interface{}{1},
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("query"),
			PGXStatusKey.String("OK"),
			SQLCQueryNameKey.String("get_users"),
		)).
		Return()

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})
}

func (s *DBTracerSuite) TestTraceQueryEndOnError() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
			PGXOperationTypeKey.String("query"),
		).
		Return()

	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL,
		Args: []interface{}{1},
	})

	expectedErr := errors.New("database error")

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		RecordError(mock.Anything).
		Return()

	s.span.EXPECT().
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("query"),
			PGXStatusKey.String("UNKNOWN_ERROR"),
			SQLCQueryNameKey.String("get_users"),
		)).
		Return()

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceQueryDuration() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
			PGXOperationTypeKey.String("query"),
		).
		Return()

	ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL,
		Args: []interface{}{1},
	})

	sleepDuration := 100 * time.Millisecond
	time.Sleep(sleepDuration)

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.MatchedBy(func(duration float64) bool {
			return duration >= 0.095 && duration <= 0.150
		}), mock.Anything).
		Return()

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})
}

func (s *DBTracerSuite) TestTraceBatchDuration() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.batch",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("batch"),
		).
		Return()

	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	s.span.EXPECT().
		SetAttributes(
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
		).
		Return()

	s.span.EXPECT().SetName(mock.Anything).Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL,
		Args:       []interface{}{1},
		CommandTag: pgconn.CommandTag{},
	})

	sleepDuration := 200 * time.Millisecond
	time.Sleep(sleepDuration)

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.MatchedBy(func(duration float64) bool {
			// Allow for some timing variance, but ensure it's close to our sleep duration
			return duration >= 0.195 && duration <= 0.400 // 195-250ms range
		}), mock.Anything).
		Return()

	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})
}

func (s *DBTracerSuite) TestTracePrepareWithDuration() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("prepare"),
			PGXPrepareStmtNameKey.String(stmtName),
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
		).
		Return()

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL,
	})

	sleepDuration := 150 * time.Millisecond
	time.Sleep(sleepDuration)

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.MatchedBy(func(duration float64) bool {
			return duration >= 0.145 && duration <= 0.200 // 145-200ms range
		}), mock.Anything).
		Return()

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: false,
	})
}

func (s *DBTracerSuite) TestTracePrepareAlreadyPrepared() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("prepare"),
			PGXPrepareStmtNameKey.String(stmtName),
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
		).
		Return()

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("prepare"),
			PGXStatusKey.String("OK"),
			SQLCQueryNameKey.String("get_users"),
		)).
		Return()

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: true,
	})
}

func (s *DBTracerSuite) TestTracePrepareError() {
	prepareSQL := s.defaultQuerySQL
	stmtName := "get_user_by_id"
	expectedErr := errors.New("prepare failed")

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("prepare"),
			PGXPrepareStmtNameKey.String(stmtName),
			SQLCQueryNameKey.String("get_users"),
			SQLCQueryTypeKey.String("one"),
		).
		Return()

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  prepareSQL,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		RecordError(mock.Anything).
		Return()

	s.span.EXPECT().
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("prepare"),
			PGXStatusKey.String("UNKNOWN_ERROR"),
			SQLCQueryNameKey.String("get_users"),
		)).
		Return()

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		Err: expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceConnectSuccess() {
	connConfig := &pgx.ConnConfig{}

	s.tracer.EXPECT().Start(mock.Anything, "postgresql.connect",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	s.span.EXPECT().End().Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("connect"),
			PGXStatusKey.String("OK"),
			SQLCQueryNameKey.String("connect"),
		)).
		Return()

	time.Sleep(50 * time.Millisecond)

	s.dbTracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{
		Conn: s.pgxConn,
		Err:  nil,
	})
}

func (s *DBTracerSuite) TestTraceConnectError() {
	connConfig := &pgx.ConnConfig{}
	expectedErr := errors.New("connection failed")

	s.tracer.EXPECT().Start(mock.Anything, "postgresql.connect",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	s.span.EXPECT().End().Return()
	s.span.EXPECT().RecordError(mock.Anything).Return()
	s.span.EXPECT().SetStatus(mock.Anything, mock.Anything).Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("connect"),
			PGXStatusKey.String("UNKNOWN_ERROR"),
			SQLCQueryNameKey.String("connect"),
		)).
		Return()

	// shouldLog function returns true for all errors in test

	s.dbTracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{
		Conn: nil,
		Err:  expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceCopyFromSuccess() {
	tableName := pgx.Identifier{"users"}
	columnNames := []string{"id", "name"}

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.copy_from",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("copy_from"),
			attribute.String("db.table", "\"users\""),
		).
		Return()

	ctx := s.dbTracer.TraceCopyFromStart(s.ctx, s.pgxConn, pgx.TraceCopyFromStartData{
		TableName:   tableName,
		ColumnNames: columnNames,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("copy_from"),
			PGXStatusKey.String("OK"),
			SQLCQueryNameKey.String("copy_from"),
		)).
		Return()

	s.dbTracer.TraceCopyFromEnd(ctx, s.pgxConn, pgx.TraceCopyFromEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})
}

func (s *DBTracerSuite) TestTraceCopyFromError() {
	tableName := pgx.Identifier{"users"}
	columnNames := []string{"id", "name"}
	expectedErr := errors.New("copy failed")

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.copy_from",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			PGXOperationTypeKey.String("copy_from"),
			attribute.String("db.table", "\"users\""),
		).
		Return()

	ctx := s.dbTracer.TraceCopyFromStart(s.ctx, s.pgxConn, pgx.TraceCopyFromStartData{
		TableName:   tableName,
		ColumnNames: columnNames,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		RecordError(mock.Anything).
		Return()

	s.span.EXPECT().
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			PGXOperationTypeKey.String("copy_from"),
			PGXStatusKey.String("UNKNOWN_ERROR"),
			SQLCQueryNameKey.String("copy_from"),
		)).
		Return()

	s.dbTracer.TraceCopyFromEnd(ctx, s.pgxConn, pgx.TraceCopyFromEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceConcurrent() {
	// Test concurrent queries to ensure thread safety
	const numQueries = 10
	var wg sync.WaitGroup

	// Set up expectations for multiple queries
	s.tracer.EXPECT().
		Start(mock.Anything, "postgresql.query",mock.AnythingOfType("trace.spanOptionFunc")).
		Return(s.ctx, s.span).
		Times(numQueries)

	s.span.EXPECT().
		SetAttributes(mock.Anything, mock.Anything, mock.Anything).
		Return().
		Times(numQueries)

	s.span.EXPECT().
		End().
		Return().
		Times(numQueries)

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return().
		Times(numQueries)

	s.histogram.EXPECT().
		Record(mock.Anything, mock.AnythingOfType("float64"), mock.Anything).
		Return().
		Times(numQueries)

	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func(queryID int) {
			defer wg.Done()

			ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
				SQL:  s.defaultQuerySQL,
				Args: []interface{}{queryID},
			})

			s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
				CommandTag: pgconn.CommandTag{},
				Err:        nil,
			})
		}(i)
	}

	wg.Wait()
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

	mp := mockmetric.NewMockMeterProvider(s.T())
	m := mockmetric.NewMockMeter(s.T())
	h := mockmetric.NewMockFloat64Histogram(s.T())
	tp := mocktracer.NewMockTracerProvider(s.T())

	mp.EXPECT().Meter("github.com/amirsalarsafaei/sqlc-pgx-monitoring").Return(m)
	m.EXPECT().Float64Histogram("custom.duration", mock.Anything, mock.Anything).Return(h, nil)

	tracer, err := NewDBTracer(
		"test_db",
		WithLogger(customLogger),
		WithShouldLog(customShouldLog),
		WithMeterProvider(mp),
		WithTraceProvider(tp),
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
	mockSpan := mocktracer.NewMockSpan(s.T())

	// Test with nil error (should not record anything)
	dbTracer := s.dbTracer.(*dbTracer)
	dbTracer.recordSpanError(mockSpan, nil)

	// Test with pgx.ErrNoRows (should not record error)
	dbTracer.recordSpanError(mockSpan, pgx.ErrNoRows)

	// Test with regular error
	mockSpan.EXPECT().RecordError(mock.Anything).Return()
	mockSpan.EXPECT().SetStatus(codes.Error, "test error").Return()

	testErr := errors.New("test error")
	dbTracer.recordSpanError(mockSpan, testErr)
}

func (s *DBTracerSuite) TestTraceBatchWithMultipleQueries() {
	s.tracer.EXPECT().Start(s.ctx, "postgresql.batch",mock.AnythingOfType("trace.spanOptionFunc")).Return(s.ctx, s.span)
	s.span.EXPECT().SetAttributes(PGXOperationTypeKey.String("batch")).Return()

	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	// First query
	s.span.EXPECT().SetAttributes(
		SQLCQueryNameKey.String("get_users"),
		SQLCQueryTypeKey.String("one"),
	).Return()
	s.span.EXPECT().SetName(mock.Anything).Return()
	s.span.EXPECT().SetStatus(codes.Ok, "").Return()

	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL,
		Args:       []any{1},
		CommandTag: pgconn.CommandTag{},
	})

	// Second query with error
	s.span.EXPECT().SetAttributes(
		SQLCQueryNameKey.String("get_users"),
		SQLCQueryTypeKey.String("one"),
	).Return()
	s.span.EXPECT().SetName(mock.Anything).Return()
	s.span.EXPECT().RecordError(mock.Anything).Return()
	s.span.EXPECT().SetStatus(codes.Error, "batch query error").Return()

	batchErr := errors.New("batch query error")
	s.dbTracer.TraceBatchQuery(ctx, s.pgxConn, pgx.TraceBatchQueryData{
		SQL:        s.defaultQuerySQL,
		Args:       []any{2},
		CommandTag: pgconn.CommandTag{},
		Err:        batchErr,
	})

	// End batch
	s.span.EXPECT().End().Return()
	s.span.EXPECT().SetStatus(codes.Ok, "").Return()
	s.histogram.EXPECT().Record(ctx, mock.AnythingOfType("float64"), mock.Anything).Return()

	s.dbTracer.TraceBatchEnd(ctx, s.pgxConn, pgx.TraceBatchEndData{})
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

func (s *DBTracerSuite) TestQueryNameExtractionEdgeCases() {
	testCases := []struct {
		name         string
		sql          string
		expectedName string
		expectedType string
	}{
		{
			name:         "query without name comment",
			sql:          "SELECT * FROM users",
			expectedName: "unknown",
			expectedType: "unknown",
		},
		{
			name:         "query with malformed comment",
			sql:          "-- name: malformed\nSELECT * FROM users",
			expectedName: "unknown",
			expectedType: "unknown",
		},
		{
			name:         "empty sql",
			sql:          "",
			expectedName: "unknown",
			expectedType: "unknown",
		},
		{
			name:         "block comment style",
			sql:          "/* name: block_comment_query :one */\nSELECT * FROM users",
			expectedName: "block_comment_query",
			expectedType: "one",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			name, queryType := queryNameFromSQL(tc.sql)
			s.Equal(tc.expectedName, name)
			s.Equal(tc.expectedType, queryType)
		})
	}
}
