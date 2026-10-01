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

var pgxOperationConnect = PGXOperationTypeKey.String("connect")

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	ctx, span := dt.startSpan(ctx, "postgresql.connect", pgxOperationConnect)

	return context.WithValue(ctx, dbTracerConnectCtxKey, &traceConnectData{
		span:       span,
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	traceData, ok := ctx.Value(dbTracerConnectCtxKey).(*traceConnectData)
	if !ok || traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "connect", nil, interval, data.Err)
	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	var logAttrs []slog.Attr
	if cfg := traceData.connConfig; cfg != nil {
		logAttrs = append(logAttrs,
			slog.String("host", cfg.Host),
			slog.Uint64("port", uint64(cfg.Port)),
			slog.String("database", cfg.Database),
		)
	}
	logAttrs = append(logAttrs, slog.Duration("time", interval))

	dt.log(ctx, "database connect", data.Err, logAttrs...)
}
