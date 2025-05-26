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
	args      []any      // 24 bytes
	span      trace.Span // 16 bytes
	sql       string     // 16 bytes
	queryName string     // 16 bytes
	queryType string     // 16 bytes
	startTime time.Time  // 8 bytes
}

func (dt *dbTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	queryName, queryType := queryNameFromSQL(data.SQL)
	ctx, span := dt.startSpan(ctx, "postgresql.query")
	span.SetAttributes(
		SQLCQueryNameKey.String(queryName),
		SQLCQueryTypeKey.String(queryType),
		PGXOperationTypeKey.String("query"),
	)

	if dt.includeQueryText {
		span.SetAttributes(semconv.DBQueryText(data.SQL))
	}

	return context.WithValue(ctx, dbTracerQueryCtxKey, &traceQueryData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
		queryName: queryName,
		queryType: queryType,
		span:      span,
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	dt.recordHistogramMetric(ctx, "query", queryData.queryName, interval, data.Err)

	defer queryData.span.End()

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(queryData.span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		queryData.span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.String("sql", queryData.sql),
			slog.String("query_name", queryData.queryName),
			slog.Any("args", dt.logQueryArgs(queryData.args)),
			slog.String("query_type", queryData.queryType),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"query",
			logAttrs...,
		)
	}
}
