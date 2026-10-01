package dbtracer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

var _ Tracer = (*dbTracer)(nil)

type dbTracer struct {
	logger          *slog.Logger
	shouldLog       ShouldLog
	logArgs         bool
	logArgsLenLimit int

	dbOperationsHist metric.Float64Histogram

	acquireConnectionHist metric.Float64Histogram
	connAcquireCounter    metric.Int64Counter
	connReleaseCounter    metric.Int64Counter

	infoAttrs []attribute.KeyValue

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
		logArgsLenLimit: defaultLogArgsLenLimit,
		latencyHistogramConfig: latencyHistogramConfig{
			description:      semconv.DBClientOperationDurationDescription,
			unit:             semconv.DBClientOperationDurationUnit,
			name:             semconv.DBClientOperationDurationName,
			bucketBoundaries: defaultBucketBoundaries,
		},
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(&optCtx)
	}

	meter := optCtx.meterProvider.Meter(optCtx.name)
	histogram, err := meter.Float64Histogram(
		optCtx.latencyHistogramConfig.name,
		metric.WithDescription(optCtx.latencyHistogramConfig.description),
		metric.WithUnit(optCtx.latencyHistogramConfig.unit),
		metric.WithExplicitBucketBoundaries(optCtx.latencyHistogramConfig.bucketBoundaries...),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing histogram meter: %w", err)
	}

	connAcquireHist, err := meter.Float64Histogram(
		"pgx.pool.trace.acquire.duration",
		metric.WithDescription("time taken to acquire a connection from the pool"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(defaultBucketBoundaries...),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing trace connection acquire meter: %w", err)
	}

	connAcquireMeter, err := meter.Int64Counter(
		"pgx.pool.trace.acquire.count",
		metric.WithDescription("counter for acquiring connections from the pool"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing trace connection acquire meter: %w", err)
	}

	connReleaseMeter, err := meter.Int64Counter(
		"pgx.pool.trace.release.count",
		metric.WithDescription("counter for releasing connections back to the pool"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing trace connection release meter: %w", err)
	}

	infoSet := attribute.NewSet(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(databaseName),
	)

	return &dbTracer{
		logger:                optCtx.logger,
		shouldLog:             optCtx.shouldLog,
		logArgs:               optCtx.logArgs,
		logArgsLenLimit:       optCtx.logArgsLenLimit,
		dbOperationsHist:      histogram,
		traceProvider:         optCtx.traceProvider,
		traceLibraryName:      optCtx.name,
		includeQueryText:      optCtx.includeSQLText,
		includeSpanNameSuffix: optCtx.includeSpanNameSuffix,
		infoAttrs:             infoSet.ToSlice(),
		acquireConnectionHist: connAcquireHist,
		connAcquireCounter:    connAcquireMeter,
		connReleaseCounter:    connReleaseMeter,
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

func (dt *dbTracer) startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return dt.getTracer().Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(dt.infoAttrs...),
		trace.WithAttributes(attrs...),
	)
}

func sqlcAttributes(qMD *queryMetadata) []attribute.KeyValue {
	if qMD == nil {
		return nil
	}

	return []attribute.KeyValue{
		SQLCQueryNameKey.String(qMD.name),
		SQLCQueryCommandKey.String(qMD.command),
		semconv.DBOperationName(qMD.name),
	}
}

func (dt *dbTracer) queryAttributes(qMD *queryMetadata, sql string) []attribute.KeyValue {
	attrs := sqlcAttributes(qMD)
	if dt.includeQueryText {
		attrs = append(attrs, semconv.DBQueryText(sql))
	}

	return attrs
}

func endSpan(span trace.Span, err error) {
	defer span.End()

	if err == nil {
		span.SetStatus(codes.Ok, "")
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		span.SetAttributes(DBStatusCodeKey.String(pgErr.Code))
	}
}

func (dt *dbTracer) log(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	level := slog.LevelInfo
	if err != nil {
		level = slog.LevelError
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	dt.logger.LogAttrs(ctx, level, msg, attrs...)
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

	dt.dbOperationsHist.Record(ctx, duration.Seconds(),
		metric.WithAttributes(dt.infoAttrs...),
		metric.WithAttributes(attrs...))
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

	limit := dt.logArgsLenLimit
	if limit <= 0 {
		limit = defaultLogArgsLenLimit
	}

	logArgs := make([]any, 0, len(args))
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

	return operationName + " " + qMD.name
}

func (dt *dbTracer) getTracer() trace.Tracer {
	return dt.traceProvider.Tracer(dt.traceLibraryName)
}

func extractConnectionID(conn *pgx.Conn) uint32 {
	if conn == nil {
		return 0
	}

	pgConn := conn.PgConn()
	if pgConn == nil {
		return 0
	}

	return pgConn.PID()
}
