package dbtracer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
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
	shouldLog       *MockShouldLog
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
	s.shouldLog = NewMockShouldLog(s.T())
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

	var err error
	s.dbTracer, err = NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithShouldLog(s.shouldLog.Execute),
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
						"db_query_duration",
						metric.WithDescription("The duration of database queries by sqlc function names"),
						metric.WithUnit("s"),
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
						"db_query_duration",
						metric.WithDescription("The duration of database queries by sqlc function names"),
						metric.WithUnit("s"),
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
		Start(s.ctx, "postgresql.query").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
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
		Start(s.ctx, "postgresql.query").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
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
			attribute.String("operation", "query"),
			attribute.String("query_name", "get_users"),
			attribute.String("query_type", "one"),
			attribute.Bool("error", false),
		)).
		Return()

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        nil,
	})
}

func (s *DBTracerSuite) TestTraceQueryDuration() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
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
		Start(s.ctx, "postgresql.batch").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "batch"),
		).
		Return()

	ctx := s.dbTracer.TraceBatchStart(s.ctx, s.pgxConn, pgx.TraceBatchStartData{})

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
		).
		Return()

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
		Start(s.ctx, "postgresql.prepare").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "prepare"),
			attribute.String("db.prepared_statement_name", stmtName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
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

func (s *DBTracerSuite) TestTraceConnectSuccess() {
	connConfig := &pgx.ConnConfig{}

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			attribute.String("operation", "connect"),
			attribute.Bool("error", false),
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

	ctx := s.dbTracer.TraceConnectStart(s.ctx, pgx.TraceConnectStartData{
		ConnConfig: connConfig,
	})
	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), s.matchAttributes(
			attribute.String("operation", "connect"),
			attribute.Bool("error", true),
		)).
		Return()

	s.shouldLog.EXPECT().
		Execute(expectedErr).
		Return(true)

	s.dbTracer.TraceConnectEnd(ctx, pgx.TraceConnectEndData{
		Conn: nil,
		Err:  expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceCopyFromSuccess() {
	tableName := pgx.Identifier{"users"}
	columnNames := []string{"id", "name"}

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.copy_from").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "copy"),
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
			attribute.String("operation", "copy"),
			attribute.String("table", "\"users\""),
			attribute.Bool("error", false),
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
		Start(s.ctx, "postgresql.copy_from").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "copy"),
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
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.span.EXPECT().
		RecordError(expectedErr).
		Return()

	s.shouldLog.EXPECT().
		Execute(expectedErr).
		Return(true)

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), metric.WithAttributes(
			attribute.String("operation", "copy"),
			attribute.String("table", "\"users\""),
			attribute.Bool("error", true),
		)).
		Return()

	s.dbTracer.TraceCopyFromEnd(ctx, s.pgxConn, pgx.TraceCopyFromEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})
}

func (s *DBTracerSuite) TestTracePrepareAlreadyPrepared() {
	stmtName := "get_user_by_id"

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "prepare"),
			attribute.String("db.prepared_statement_name", stmtName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
		).
		Return()

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  s.defaultQuerySQL,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), metric.WithAttributes(
			attribute.String("operation", "prepare"),
			attribute.String("statement_name", stmtName),
			attribute.String("query_name", "get_users"),
			attribute.String("query_type", "one"),
			attribute.Bool("error", false),
		)).
		Return()

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		AlreadyPrepared: true,
	})
}

func (s *DBTracerSuite) TestTracePrepareError() {
	stmtName := "get_user_by_id"
	expectedErr := errors.New("prepare failed")

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "prepare"),
			attribute.String("db.prepared_statement_name", stmtName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
		).
		Return()

	ctx := s.dbTracer.TracePrepareStart(s.ctx, s.pgxConn, pgx.TracePrepareStartData{
		Name: stmtName,
		SQL:  s.defaultQuerySQL,
	})

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.span.EXPECT().
		RecordError(expectedErr).
		Return()

	s.shouldLog.EXPECT().
		Execute(expectedErr).
		Return(true)

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), metric.WithAttributes(
			attribute.String("operation", "prepare"),
			attribute.String("statement_name", stmtName),
			attribute.String("query_name", "get_users"),
			attribute.String("query_type", "one"),
			attribute.Bool("error", true),
		)).
		Return()

	s.dbTracer.TracePrepareEnd(ctx, s.pgxConn, pgx.TracePrepareEndData{
		Err: expectedErr,
	})
}

func (s *DBTracerSuite) TestTraceConcurrent() {
	numGoroutines := 10
	var wg sync.WaitGroup
	results := make(chan error, numGoroutines)

	trigger := make(chan int)

	s.tracer.EXPECT().
		Start(mock.Anything, "postgresql.query").
		Return(s.ctx, s.span).
		Times(numGoroutines)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
		).
		Return().
		Times(numGoroutines)

	s.span.EXPECT().
		End().
		Return().
		Times(numGoroutines)

	s.span.EXPECT().
		SetStatus(codes.Ok, "").
		Return().
		Times(numGoroutines)

	s.histogram.EXPECT().
		Record(mock.Anything, mock.AnythingOfType("float64"), mock.Anything).
		Return().
		Times(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int, trigger <-chan int) {
			defer wg.Done()

			ctx := s.dbTracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
				SQL:  s.defaultQuerySQL,
				Args: []interface{}{id},
			})

			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

			s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
				CommandTag: pgconn.CommandTag{},
				Err:        nil,
			})

			results <- nil
		}(i, trigger)
	}

	time.Sleep(500 * time.Millisecond)
	close(trigger)
	wg.Wait()
	close(results)

	var errs []error
	for err := range results {
		if err != nil {
			errs = append(errs, err)
		}
	}

	s.NoError(errors.Join(errs...), "Expected no errors in concurrent execution")
}

func (s *DBTracerSuite) TestLoggerBehavior() {
	var logBuffer bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tracer, err := NewDBTracer(
		s.defaultDBName,
		WithTraceProvider(s.tracerProvider),
		WithMeterProvider(s.meterProvider),
		WithLogger(customLogger),
		WithShouldLog(func(err error) bool { return true }),
	)
	s.Require().NoError(err)

	// Setup for query execution
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
		).
		Return()

	ctx := tracer.TraceQueryStart(s.ctx, s.pgxConn, pgx.TraceQueryStartData{
		SQL:  s.defaultQuerySQL,
		Args: []interface{}{1},
	})

	expectedErr := errors.New("test error code:9123")

	s.span.EXPECT().
		End().
		Return()

	s.span.EXPECT().
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.span.EXPECT().
		RecordError(expectedErr).
		Return()

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), mock.Anything).
		Return()

	tracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})

	logOutput := logBuffer.String()
	s.Contains(logOutput, "test error code:9123")
	s.Contains(logOutput, "get_users")
	s.Contains(logOutput, "Query failed")
}

func (s *DBTracerSuite) TestTraceQueryEndOnError() {
	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.query").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.query_name", "get_users"),
			attribute.String("db.query_type", "one"),
			attribute.String("db.operation", "query"),
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
		SetStatus(codes.Error, expectedErr.Error()).
		Return()

	s.span.EXPECT().
		RecordError(expectedErr).
		Return()

	s.shouldLog.EXPECT().
		Execute(expectedErr).
		Return(true)

	s.histogram.EXPECT().
		Record(ctx, mock.AnythingOfType("float64"), mock.Anything).
		Return()

	s.dbTracer.TraceQueryEnd(ctx, s.pgxConn, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        expectedErr,
	})
}
