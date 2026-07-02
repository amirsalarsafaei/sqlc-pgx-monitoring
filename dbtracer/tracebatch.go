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
	startTime       time.Time // 24 bytes
	batchQuerySpans []trace.Span
	qMD             *queryMetadata
	batchIndex      int
}

var (
	pgxOperationBatch      = PGXOperationTypeKey.String("batch")
	pgxOperationBatchQuery = PGXOperationTypeKey.String("batch.query")
)

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, batch pgx.TraceBatchStartData) context.Context {
	// sqlc does not allow mixing different queries in the same batch, so every
	// queued query shares the same operation name. We derive it from the first
	// query and use it to name the batch span, matching the per-query spans.
	var batchQMD *queryMetadata
	if batch.Batch != nil && len(batch.Batch.QueuedQueries) > 0 {
		batchQMD = queryMetadataFromSQL(batch.Batch.QueuedQueries[0].SQL)
	}

	ctx, span := dt.getTracer().Start(ctx, dt.spanName("postgresql.batch", batchQMD),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(dt.infoAttrs...),
		trace.WithAttributes(pgxOperationBatch))

	if batchQMD != nil {
		span.SetAttributes(
			SQLCQueryNameKey.String(batchQMD.name),
			SQLCQueryCommandKey.String(batchQMD.command),
			semconv.DBOperationName(batchQMD.name),
		)
	}

	var batchQuerySpans []trace.Span
	if batch.Batch != nil {
		batchQuerySpans = make([]trace.Span, len(batch.Batch.QueuedQueries))
		for i, q := range batch.Batch.QueuedQueries {
			qMD := queryMetadataFromSQL(q.SQL)

			_, querySpan := dt.getTracer().Start(ctx, dt.spanName("postgresql.batch.query", qMD),
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(dt.infoAttrs...),
				trace.WithAttributes(pgxOperationBatchQuery))

			if qMD != nil {
				querySpan.SetAttributes(
					SQLCQueryNameKey.String(qMD.name),
					SQLCQueryCommandKey.String(qMD.command),
					semconv.DBOperationName(qMD.name),
				)
			}

			batchQuerySpans[i] = querySpan
		}
	}

	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime:       time.Now(),
		batchQuerySpans: batchQuerySpans,
		qMD:             batchQMD,
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

	dt.recordDBOperationHistogramMetric(ctx, "batch", traceData.qMD, interval, data.Err)

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
