package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

type traceConnectData struct {
	span       trace.Span
	startTime  time.Time
	connConfig *pgx.ConnConfig
}

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	ctx, span := dt.startSpan(ctx, "postgresql.connect")

	return context.WithValue(ctx, dbTracerConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
		span:       span,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	connectData := ctx.Value(dbTracerConnectCtxKey).(*traceConnectData)

	endTime := time.Now()
	interval := endTime.Sub(connectData.startTime)

	dt.recordHistogramMetric(ctx, "connect", "connect", interval, data.Err)

	defer connectData.span.End()

	if data.Err != nil {
		dt.recordSpanError(connectData.span, data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"database connect",
				slog.String("host", connectData.connConfig.Host),
				slog.Uint64("port", uint64(connectData.connConfig.Port)),
				slog.String("database", connectData.connConfig.Database),
				slog.Duration("time", interval),
				slog.Any("error", data.Err),
			)
		}
		return
	}

	dt.logger.LogAttrs(ctx, slog.LevelInfo,
		"database connect",
		slog.String("host", connectData.connConfig.Host),
		slog.Uint64("port", uint64(connectData.connConfig.Port)),
		slog.String("database", connectData.connConfig.Database),
		slog.Duration("time", interval),
	)
}
