package dbtracer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

var errBatchQueryNotExecuted = errors.New("batch query not executed")

type traceBatchData struct {
	span       trace.Span
	startTime  time.Time
	querySpans []trace.Span
	qMD        *queryMetadata
	queryIndex int
	ended      bool
}

var (
	pgxOperationBatch      = PGXOperationTypeKey.String("batch")
	pgxOperationBatchQuery = PGXOperationTypeKey.String("batch.query")
)

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	var queued []*pgx.QueuedQuery
	if data.Batch != nil {
		queued = data.Batch.QueuedQueries
	}

	// sqlc queues a single query per batch, so the first query names the batch.
	var batchQMD *queryMetadata
	if len(queued) > 0 {
		batchQMD = queryMetadataFromSQL(queued[0].SQL)
	}

	ctx, span := dt.startSpan(ctx, dt.spanName("postgresql.batch", batchQMD),
		append(sqlcAttributes(batchQMD), pgxOperationBatch)...)

	querySpans := make([]trace.Span, len(queued))
	for i, q := range queued {
		qMD := queryMetadataFromSQL(q.SQL)
		_, querySpans[i] = dt.startSpan(ctx, dt.spanName("postgresql.batch.query", qMD),
			append(dt.queryAttributes(qMD, q.SQL), pgxOperationBatchQuery)...)
	}

	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		span:       span,
		startTime:  time.Now(),
		querySpans: querySpans,
		qMD:        batchQMD,
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	traceData, ok := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if !ok || traceData == nil {
		return
	}

	if traceData.queryIndex >= len(traceData.querySpans) {
		return
	}

	endSpan(traceData.querySpans[traceData.queryIndex], data.Err)
	traceData.queryIndex++

	if !dt.shouldLog(data.Err) {
		return
	}

	var logAttrs []slog.Attr
	if data.Err == nil {
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
	}
	logAttrs = append(logAttrs,
		slog.String("sql", data.SQL),
		slog.Any("args", dt.logQueryArgs(data.Args)),
		slog.Uint64("pid", uint64(extractConnectionID(conn))),
	)

	dt.log(ctx, "batch query", data.Err, logAttrs...)
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	traceData, ok := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	// pgx calls TraceBatchEnd twice when SendBatch fails early.
	if !ok || traceData == nil || traceData.ended {
		return
	}
	traceData.ended = true

	interval := time.Since(traceData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "batch", traceData.qMD, interval, data.Err)

	// pgx stops reporting queued queries after the first failure.
	if pending := traceData.querySpans[traceData.queryIndex:]; len(pending) > 0 {
		pendingErr := data.Err
		if pendingErr == nil {
			pendingErr = errBatchQueryNotExecuted
		}
		for _, span := range pending {
			endSpan(span, pendingErr)
		}
		traceData.queryIndex = len(traceData.querySpans)
	}

	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	dt.log(ctx, "batch end", data.Err,
		slog.Duration("interval", interval),
		slog.Uint64("pid", uint64(extractConnectionID(conn))),
	)
}
