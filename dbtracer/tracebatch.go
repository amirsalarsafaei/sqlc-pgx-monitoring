package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type traceBatchData struct {
	startTime       time.Time // 16 bytes
	batchQuerySpans []trace.Span
	batchIndex      int
}

var (
	pgxOperationBatch      = PGXOperationTypeKey.String("batch")
	pgxOperationBatchQuery = PGXOperationTypeKey.String("batch.query")
)

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, batch pgx.TraceBatchStartData) context.Context {
	ctx, _ = dt.getTracer().Start(ctx, "postgresql.batch", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			dt.infoAttrs...,
		), trace.WithAttributes(pgxOperationBatch))

	var batchQuerySpans []trace.Span
	if batch.Batch != nil {
		batchQuerySpans = make([]trace.Span, len(batch.Batch.QueuedQueries))
		for i, q := range batch.Batch.QueuedQueries {
			_, span := dt.getTracer().Start(ctx, "postgresql.batch.query", trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(dt.infoAttrs...), trace.WithAttributes(
					pgxOperationBatchQuery))

			qMD := queryMetadataFromSQL(q.SQL)
			if qMD != nil {
				span.SetAttributes(
					SQLCQueryNameKey.String(qMD.name),
					SQLCQueryCommandKey.String(qMD.command),
					semconv.DBOperationName(qMD.name),
				)
			}

			batchQuerySpans[i] = span
		}
	}

	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime:       time.Now(),
		batchQuerySpans: batchQuerySpans,
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	traceData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if traceData == nil {
		return
	}

	if traceData.batchIndex >= len(traceData.batchQuerySpans) {
		return
	}

	span := traceData.batchQuerySpans[traceData.batchIndex]
	defer span.End()
	traceData.batchIndex++

	var logAttrs []slog.Attr
	var level slog.Level
	if data.Err != nil {
		span.SetStatus(codes.Error, data.Err.Error())
		span.RecordError(data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.String("sql", data.SQL),
			slog.Any("args", dt.logQueryArgs(data.Args)),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"batch query",
			logAttrs...,
		)
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	traceData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if traceData == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	defer span.End()

	interval := time.Since(traceData.startTime)

	dt.recordDBOperationHistogramMetric(ctx, "batch", nil, interval, data.Err)

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		span.SetStatus(codes.Ok, "")
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
