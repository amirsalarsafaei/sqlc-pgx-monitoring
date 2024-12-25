package dbtracer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
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

func (dt *dbTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	queryName, queryType := queryNameFromSQL(data.SQL)
	ctx, span := dt.tracer.Start(ctx, "postgresql.query")
	span.SetAttributes(
		attribute.String("db.name", dt.databaseName),
		attribute.String("db.query_name", queryName),
		attribute.String("db.query_type", queryType),
		attribute.String("db.operation", "query"),
	)
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
	histogramAttrs := []attribute.KeyValue{
		attribute.String("operation", "query"),
		attribute.String("query_name", queryData.queryName),
		attribute.String("query_type", queryData.queryType),
		attribute.Bool("error", data.Err != nil),
	}
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(histogramAttrs...))

	defer queryData.span.End()

	if data.Err != nil {
		queryData.span.SetStatus(codes.Error, data.Err.Error())
		queryData.span.RecordError(data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				fmt.Sprintf("Query: %s", queryData.queryName),
				slog.String("sql", queryData.sql),
				slog.Any("args", dt.logQueryArgs(queryData.args)),
				slog.String("query_type", queryData.queryType),
				slog.Duration("time", interval),
				slog.Uint64("pid", uint64(extractConnectionID(conn))),
				slog.String("error", data.Err.Error()),
			)
		}
	} else {
		queryData.span.SetStatus(codes.Ok, "")
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			fmt.Sprintf("Query: %s", queryData.queryName),
			slog.String("sql", queryData.sql),
			slog.String("query_type", queryData.queryType),
			slog.Any("args", dt.logQueryArgs(queryData.args)),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.String("commandTag", data.CommandTag.String()),
		)
	}
}
