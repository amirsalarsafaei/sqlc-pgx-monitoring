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
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	pgxpool.AcquireTracer
	pgxpool.ReleaseTracer
}

// dbTracer implements pgx.QueryTracer, pgx.BatchTracer, pgx.ConnectTracer, and pgx.CopyFromTracer
type dbTracer struct {
	logger          *slog.Logger
	shouldLog       ShouldLog
	databaseName    string
	logArgs         bool
	logArgsLenLimit int

	dbOperationsHist metric.Float64Histogram

	poolOperationsHist metric.Float64Histogram
	connAcquireCounter metric.Int64Counter
	connReleaseCounter metric.Int64Counter

	infoAttrs []attribute.KeyValue
	traceSetupConfig trace.SpanConfig

	traceProvider         trace.TracerProvider
	traceLibraryName      string
	includeQueryText      bool
	includeSpanNameSuffix bool
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
	infoSet := attribute.NewSet(
			semconv.DBSystemPostgreSQL,
			semconv.DBNamespace(databaseName),
		)
	return &dbTracer{
		logger:                optCtx.logger,
		databaseName:          databaseName,
		shouldLog:             optCtx.shouldLog,
		logArgs:               optCtx.logArgs,
		logArgsLenLimit:       optCtx.logArgsLenLimit,
		dbOperationsHist:      histogram,
		traceProvider:         optCtx.traceProvider,
		traceLibraryName:      optCtx.name,
		includeQueryText:      optCtx.includeSQLText,
		includeSpanNameSuffix: optCtx.includeSpanNameSuffix,
		infoAttrs: infoSet.ToSlice(),
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
	dbTracerAcquireCtxKey
)

func (dt *dbTracer) recordSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			span.SetAttributes(DBStatusCodeKey.String(pgErr.Code))
		}
	}
}

func (dt *dbTracer) recordDBOperationHistogramMetric(ctx context.Context,
	pgxOperation string, qMD *queryMetadata, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		PGXStatusKey.String(pgxStatusFromErr(err)),
		PGXOperationTypeKey.String(pgxOperation),
	}

	if qMD != nil {
		attrs = append(attrs, SQLCQueryNameKey.String(qMD.name),
			SQLCQueryCommandKey.String(qMD.command))
	}

	dt.dbOperationsHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
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

func (dt *dbTracer) spanName(operationName string, qMD *queryMetadata) string {
	if !dt.includeSpanNameSuffix || qMD == nil {
		return operationName
	}

	return fmt.Sprintf("%s %s", operationName, qMD.name)
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
