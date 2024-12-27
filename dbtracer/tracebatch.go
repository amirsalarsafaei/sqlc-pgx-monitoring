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

type traceBatchData struct {
	span      trace.Span // 16 bytes
	startTime time.Time  // 16 bytes
	queryName string     // 16 bytes
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	ctx, span := dt.getTracer().Start(ctx, "postgresql.batch")
	span.SetAttributes(
		attribute.String("db.name", dt.databaseName),
		attribute.String("db.operation", "batch"),
	)
	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime: time.Now(),
		span:      span,
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	queryName, queryType := queryNameFromSQL(data.SQL)
	queryData.queryName = queryName
	queryData.span.SetAttributes(
		attribute.String("db.query_name", queryName),
		attribute.String("db.query_type", queryType),
	)

	if data.Err != nil {
		queryData.span.SetStatus(codes.Error, data.Err.Error())
		queryData.span.RecordError(data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"Query",
				slog.String("sql", data.SQL),
				slog.Any("args", dt.logQueryArgs(data.Args)),
				slog.Uint64("pid", uint64(extractConnectionID(conn))),
				slog.String("error", data.Err.Error()),
			)
		}
	} else {
		queryData.span.SetStatus(codes.Ok, "")
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"Query",
			slog.String("sql", data.SQL),
			slog.Any("args", dt.logQueryArgs(data.Args)),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.String("commandTag", data.CommandTag.String()),
		)
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	defer queryData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "batch"),
		attribute.String("query_name", queryData.queryName),
		attribute.Bool("error", data.Err != nil),
	))

	if data.Err != nil {
		queryData.span.SetStatus(codes.Error, data.Err.Error())
		queryData.span.RecordError(data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				fmt.Sprintf("Query: %s", queryData.queryName),
				slog.Duration("interval", interval),
				slog.Uint64("pid", uint64(extractConnectionID(conn))),
				slog.String("error", data.Err.Error()),
			)
		}
	} else {
		queryData.span.SetStatus(codes.Ok, "")
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"Query",
			slog.Duration("interval", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)
	}
}
