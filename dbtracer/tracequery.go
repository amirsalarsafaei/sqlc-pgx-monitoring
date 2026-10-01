package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
)

type traceQueryData struct {
	span      trace.Span
	args      []any
	sql       string
	qMD       *queryMetadata
	startTime time.Time
}

var pgxOperationQuery = PGXOperationTypeKey.String("query")

func (dt *dbTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	qMD := queryMetadataFromSQL(data.SQL)

	ctx, span := dt.startSpan(ctx, dt.spanName("postgresql.query", qMD),
		append(dt.queryAttributes(qMD, data.SQL), pgxOperationQuery)...)

	return context.WithValue(ctx, dbTracerQueryCtxKey, &traceQueryData{
		span:      span,
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
		qMD:       qMD,
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	traceData, ok := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	if !ok || traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "query", traceData.qMD, interval, data.Err)
	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	var logAttrs []slog.Attr
	if data.Err == nil {
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
	}
	if traceData.qMD != nil {
		logAttrs = append(logAttrs,
			slog.String("query_name", traceData.qMD.name),
			slog.String("query_command", traceData.qMD.command),
		)
	}
	logAttrs = append(logAttrs,
		slog.String("sql", traceData.sql),
		slog.Any("args", dt.logQueryArgs(traceData.args)),
		slog.Duration("time", interval),
		slog.Uint64("pid", uint64(extractConnectionID(conn))),
	)

	dt.log(ctx, "query", data.Err, logAttrs...)
}
