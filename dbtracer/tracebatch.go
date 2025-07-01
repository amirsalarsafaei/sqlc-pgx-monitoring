package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type traceBatchData struct {
	span      trace.Span // 16 bytes
	startTime time.Time  // 16 bytes
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	spanName := dt.spanName("postgresql.batch", nil)

	ctx, span := dt.startSpan(ctx, spanName, PGXOperationTypeKey.String("batch"))

	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime: time.Now(),
		span:      span,
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	traceData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
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
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.String("sql", data.SQL),
			slog.Any("args", dt.logQueryArgs(data.Args)),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"batch",
			logAttrs...,
		)
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	traceData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if traceData == nil {
		return
	}
	defer traceData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(traceData.startTime)

	dt.recordHistogramMetric(ctx, "batch", nil, interval, data.Err)

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(traceData.span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		traceData.span.SetStatus(codes.Ok, "")
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.Duration("interval", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"batch end",
			logAttrs...,
		)
	}
}
