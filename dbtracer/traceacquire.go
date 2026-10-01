package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var pgxPoolConnOperationAcquire = PGXPoolConnOperationKey.String("acquire")

type traceAcquireData struct {
	span      trace.Span
	startTime time.Time
}

func (dt *dbTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	ctx, span := dt.startSpan(ctx, "pgxpool.acquire", pgxPoolConnOperationAcquire)

	return context.WithValue(ctx, dbTracerAcquireCtxKey, &traceAcquireData{
		span:      span,
		startTime: time.Now(),
	})
}

func (dt *dbTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	traceData, ok := ctx.Value(dbTracerAcquireCtxKey).(*traceAcquireData)
	if !ok || traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)
	status := PGXStatusKey.String(pgxStatusFromErr(data.Err))
	dt.connAcquireCounter.Add(ctx, 1,
		metric.WithAttributes(dt.infoAttrs...),
		metric.WithAttributes(pgxPoolConnOperationAcquire, status))
	dt.acquireConnectionHist.Record(ctx, interval.Seconds(),
		metric.WithAttributes(dt.infoAttrs...),
		metric.WithAttributes(status))
	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	dt.log(ctx, "acquire connection", data.Err,
		slog.Uint64("pid", uint64(extractConnectionID(data.Conn))),
	)
}
