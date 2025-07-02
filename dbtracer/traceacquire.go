package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	pgxPoolConnOperationAcquire = PGXPoolConnOperationKey.String("acquire")
)

type traceAcquireData struct {
	span      trace.Span
	startTime time.Time
}

// TraceAcquireStart implements Tracer.
func (dt *dbTracer) TraceAcquireStart(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireStartData) context.Context {
	ctx, span := dt.getTracer().Start(ctx, "pgxpool.acquire", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			dt.infoAttrs...
		), trace.WithAttributes(pgxPoolConnOperationAcquire))

	return context.WithValue(ctx, dbTracerAcquireCtxKey, &traceAcquireData{
		startTime: time.Now(),
		span:      span,
	})
}

// TraceAcquireEnd implements Tracer.
func (dt *dbTracer) TraceAcquireEnd(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	traceData := ctx.Value(dbTracerAcquireCtxKey).(*traceAcquireData)
	if traceData == nil {
		return
	}

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		traceData.span.SetStatus(codes.Error, data.Err.Error())
		traceData.span.RecordError(data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		traceData.span.SetStatus(codes.Ok, "")
		level = slog.LevelInfo
	}

	dt.connAcquireCounter.Add(ctx, 1, metric.WithAttributes(
		pgxPoolConnOperationAcquire,
		PGXStatusKey.String(pgxStatusFromErr(data.Err)),
	))

	dt.acquireConnectionHist.Record(ctx, time.Since(traceData.startTime).Seconds(), 
		metric.WithAttributes(dt.infoAttrs...),
		metric.WithAttributes(PGXStatusKey.String(pgxStatusFromErr(data.Err))))

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.Uint64("pid", uint64(extractConnectionID(data.Conn))))

		dt.logger.LogAttrs(ctx, level,
			"acquire connection",
			logAttrs...,
		)
	}
}
