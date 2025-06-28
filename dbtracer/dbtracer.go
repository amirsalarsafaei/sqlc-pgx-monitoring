package dbtracer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var ErrDatabaseNameEmpty = errors.New("database name is empty")

type Tracer interface {
	pgx.BatchTracer
	pgx.ConnectTracer
	pgx.CopyFromTracer
	pgx.QueryTracer
	pgx.PrepareTracer
}

// dbTracer implements pgx.QueryTracer, pgx.BatchTracer, pgx.ConnectTracer, and pgx.CopyFromTracer
type dbTracer struct {
	logger                *slog.Logger
	shouldLog             ShouldLog
	databaseName          string
	logArgs               bool
	logArgsLenLimit       int
	histogram             metric.Float64Histogram
	traceProvider         trace.TracerProvider
	traceLibraryName      string
	includeQueryText      bool
	appendQueryNameToSpan bool
}

func NewDBTracer(
	databaseName string,
	opts ...Option,
) (Tracer, error) {
	if databaseName == "" {
		return nil, ErrDatabaseNameEmpty
	}

	optCtx := optionCtx{
		name: "github.com/amirsalarsafaei/sqlc-pgx-monitoring",
		shouldLog: func(_ error) bool {
			return true
		},
		meterProvider:   otel.GetMeterProvider(),
		traceProvider:   otel.GetTracerProvider(),
		logArgs:         true,
		logArgsLenLimit: 64,
		latencyHistogramConfig: struct {
			name        string
			unit        string
			description string
		}{
			description: semconv.DBClientOperationDurationDescription,
			unit:        semconv.DBClientOperationDurationUnit,
			name:        semconv.DBClientOperationDurationName,
		},
		logger:         slog.Default(),
		includeSQLText: false,
	}
	for _, opt := range opts {
		opt(&optCtx)
	}

	meter := optCtx.meterProvider.Meter(optCtx.name)
	histogram, err := meter.Float64Histogram(
		optCtx.latencyHistogramConfig.name,
		metric.WithDescription(optCtx.latencyHistogramConfig.description),
		metric.WithUnit(optCtx.latencyHistogramConfig.unit),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing histogram meter: [%w]", err)
	}

	return &dbTracer{
		logger:                optCtx.logger,
		databaseName:          databaseName,
		shouldLog:             optCtx.shouldLog,
		logArgs:               optCtx.logArgs,
		logArgsLenLimit:       optCtx.logArgsLenLimit,
		histogram:             histogram,
		traceProvider:         optCtx.traceProvider,
		traceLibraryName:      optCtx.name,
		includeQueryText:      optCtx.includeSQLText,
		appendQueryNameToSpan: optCtx.appendQueryNameToSpan,
	}, nil
}

type ctxKey int

const (
	_ ctxKey = iota
	dbTracerQueryCtxKey
	dbTracerBatchCtxKey
	dbTracerCopyFromCtxKey
	dbTracerConnectCtxKey
	dbTracerPrepareCtxKey
)

func (dt *dbTracer) recordSpanError(span trace.Span, err error) {
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			span.SetAttributes(DBStatusCodeKey.String(pgErr.Code))
		}
	}
}

func (dt *dbTracer) recordHistogramMetric(ctx context.Context, pgxOperation string, queryName string, duration time.Duration, err error) {
	dt.histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
		PGXStatusKey.String(pgxStatusFromErr(err)),
		PGXOperationTypeKey.String(pgxOperation),
		SQLCQueryNameKey.String(queryName),
	))
}

func pgxStatusFromErr(err error) string {
	if err == nil {
		return "OK"
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Severity
	}

	return "UNKNOWN_ERROR"
}

func (dt *dbTracer) logQueryArgs(args []any) []any {
	if !dt.logArgs {
		return nil
	}

	logArgs := make([]any, 0, len(args))
	limit := dt.logArgsLenLimit
	if limit == 0 {
		limit = 64 // default limit if not set
	}

	for _, a := range args {
		switch v := a.(type) {
		case []byte:
			if len(v) <= limit {
				a = hex.EncodeToString(v)
			} else {
				a = fmt.Sprintf("%x (truncated %d bytes)", v[:limit], len(v)-limit)
			}
		case string:
			if len(v) > limit {
				var l int
				for w := 0; l < limit; l += w {
					_, w = utf8.DecodeRuneInString(v[l:])
				}

				if len(v) > l {
					a = fmt.Sprintf("%s (truncated %d bytes)", v[:l], len(v)-l)
				}
			}
		}

		logArgs = append(logArgs, a)
	}

	return logArgs
}

func (dt *dbTracer) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := dt.getTracer().Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(dt.databaseName),
	)

	return ctx, span
}

func (dt *dbTracer) getTracer() trace.Tracer {
	return dt.traceProvider.Tracer(dt.traceLibraryName)
}

func extractConnectionID(conn *pgx.Conn) uint32 {
	if conn == nil {
		return 0
	}

	pgConn := conn.PgConn()
	if pgConn != nil {
		pid := pgConn.PID()
		return pid
	}
	return 0
}
