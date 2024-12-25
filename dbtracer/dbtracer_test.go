package dbtracer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

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
		Return(s.tracer)

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
		name         string
		databaseName string
		opts         []Option
		setupMocks   func(*mockmetric.MockMeterProvider, *mockmetric.MockMeter, *mockmetric.MockFloat64Histogram)
		wantErr      bool
	}{
		{
			name:         "successful creation",
			databaseName: "test_db",
			opts:         []Option{},
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram) {
				mp.EXPECT().
					Meter(mock.Anything).
					Return(m)
				m.EXPECT().
					Float64Histogram(mock.Anything, mock.Anything, mock.Anything).
					Return(h, nil)
			},
			wantErr: false,
		}, {
			name:         "meter creation failure",
			databaseName: "test_db",
			opts:         []Option{},
			setupMocks: func(mp *mockmetric.MockMeterProvider, m *mockmetric.MockMeter, h *mockmetric.MockFloat64Histogram) {
				mp.EXPECT().
					Meter(mock.Anything).
					Return(m)
				m.EXPECT().
					Float64Histogram(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("meter creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			mp := mockmetric.NewMockMeterProvider(s.T())
			m := mockmetric.NewMockMeter(s.T())
			h := mockmetric.NewMockFloat64Histogram(s.T())

			tt.setupMocks(mp, m, h)

			opts := append(tt.opts, WithMeterProvider(mp))
			_, err := NewDBTracer(tt.databaseName, opts...)

			if tt.wantErr {
				s.Error(err)
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
		Record(ctx, mock.AnythingOfType("float64"), mock.Anything).
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
	prepareSQL := "SELECT * FROM users WHERE id = $1"
	stmtName := "get_user_by_id"

	s.tracer.EXPECT().
		Start(s.ctx, "postgresql.prepare").
		Return(s.ctx, s.span)

	s.span.EXPECT().
		SetAttributes(
			attribute.String("db.name", s.defaultDBName),
			attribute.String("db.operation", "prepare"),
			attribute.String("db.prepared_statement_name", stmtName),
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

func (s *DBTracerSuite) TestTraceQueryEnd_Error() {
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
