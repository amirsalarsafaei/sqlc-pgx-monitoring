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

type traceQueryData struct {
	args      []any  // 24 bytes
	sql       string // 16 bytes
	qMD       *queryMetadata
	startTime time.Time // 8 bytes
}

var pgxOperationQuery = PGXOperationTypeKey.String("query")

func (dt *dbTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	qMD := queryMetadataFromSQL(data.SQL)

	spanName := dt.spanName("postgresql.query", qMD)

	ctx, span := dt.getTracer().Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			dt.infoAttrs...),
		trace.WithAttributes(pgxOperationQuery),
	)

	if qMD != nil {
		span.SetAttributes(
			SQLCQueryNameKey.String(qMD.name),
			SQLCQueryCommandKey.String(qMD.command),
			semconv.DBOperationName(qMD.name),
		)
	}

	if dt.includeQueryText {
		span.SetAttributes(semconv.DBQueryText(data.SQL))
	}

	return context.WithValue(ctx, dbTracerQueryCtxKey, &traceQueryData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
		qMD:       qMD,
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	traceData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)
	if traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)

	dt.recordDBOperationHistogramMetric(ctx, "query", traceData.qMD, interval, data.Err)

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	defer span.End()

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		if traceData.qMD != nil {
			logAttrs = append(logAttrs,
				slog.String("query_name", traceData.qMD.name),
				slog.String("query_command", traceData.qMD.command),
			)
		}

		logAttrs = append(logAttrs, slog.String("sql", traceData.sql),
			slog.Any("args", dt.logQueryArgs(traceData.args)),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"query",
			logAttrs...,
		)
	}
}
