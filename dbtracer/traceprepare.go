package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

type tracePrepareData struct {
	span          trace.Span
	startTime     time.Time
	qMD           *queryMetadata
	sql           string
	statementName string
}

var pgxOperationPrepare = PGXOperationTypeKey.String("prepare")

func (dt *dbTracer) TracePrepareStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	qMD := queryMetadataFromSQL(data.SQL)

	ctx, span := dt.startSpan(ctx, dt.spanName("postgresql.prepare", qMD),
		append(dt.queryAttributes(qMD, data.SQL), pgxOperationPrepare, PGXPrepareStmtNameKey.String(data.Name))...)

	return context.WithValue(ctx, dbTracerPrepareCtxKey, &tracePrepareData{
		span:          span,
		startTime:     time.Now(),
		statementName: data.Name,
		sql:           data.SQL,
		qMD:           qMD,
	})
}

func (dt *dbTracer) TracePrepareEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareEndData,
) {
	traceData, ok := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)
	if !ok || traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "prepare", traceData.qMD, interval, data.Err)
	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	var logAttrs []slog.Attr
	if data.Err == nil {
		logAttrs = append(logAttrs, slog.Bool("alreadyPrepared", data.AlreadyPrepared))
	}
	logAttrs = append(logAttrs,
		slog.String("statement_name", traceData.statementName),
		slog.String("sql", traceData.sql),
		slog.Duration("time", interval),
		slog.Uint64("pid", uint64(extractConnectionID(conn))),
	)

	dt.log(ctx, "prepare", data.Err, logAttrs...)
}
