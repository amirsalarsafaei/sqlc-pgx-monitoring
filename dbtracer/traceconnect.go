package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type traceConnectData struct {
	startTime  time.Time
	connConfig *pgx.ConnConfig
}

var pgxOperationConnect = PGXOperationTypeKey.String("connect")

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {

	ctx, _ = dt.getTracer().Start(ctx, "postgresql.connect", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			dt.infoAttrs...),
		trace.WithAttributes(pgxOperationConnect))

	return context.WithValue(ctx, dbTracerConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	traceData := ctx.Value(dbTracerConnectCtxKey).(*traceConnectData)
	if traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)

	dt.recordDBOperationHistogramMetric(ctx, "connect", nil, interval, data.Err)

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	defer span.End()

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(span, data.Err)
		logAttrs = append(logAttrs, slog.Any("error", data.Err))
		level = slog.LevelError
	} else {
		span.SetStatus(codes.Ok, "")
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs,
			slog.String("host", traceData.connConfig.Host),
			slog.Uint64("port", uint64(traceData.connConfig.Port)),
			slog.String("database", traceData.connConfig.Database),
			slog.Duration("time", interval),
		)

		dt.logger.LogAttrs(ctx, level,
			"database connect",
			logAttrs...,
		)
	}
}
