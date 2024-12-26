package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type traceConnectData struct {
	startTime  time.Time
	connConfig *pgx.ConnConfig
}

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	return context.WithValue(ctx, dbTracerConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	connectData := ctx.Value(dbTracerConnectCtxKey).(*traceConnectData)

	endTime := time.Now()
	interval := endTime.Sub(connectData.startTime)

	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "connect"),
		attribute.Bool("error", data.Err != nil),
	))

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"database connect",
				slog.String("host", connectData.connConfig.Host),
				slog.Uint64("port", uint64(connectData.connConfig.Port)),
				slog.String("database", connectData.connConfig.Database),
				slog.Duration("time", interval),
				slog.String("error", data.Err.Error()),
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

	if data.Conn != nil {
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"database connect",
			slog.String("host", connectData.connConfig.Host),
			slog.Uint64("port", uint64(connectData.connConfig.Port)),
			slog.String("database", connectData.connConfig.Database),
			slog.Duration("time", interval),
		)
	}
}
