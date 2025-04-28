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
	span      trace.Span // 16 bytes
	startTime time.Time  // 16 bytes
	queryName string     // 16 bytes
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	ctx, span := dt.startSpan(ctx, "postgresql.batch")
	span.SetAttributes(
		PGXOperationTypeKey.String("batch"),
	)
	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime: time.Now(),
		span:      span,
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if queryData == nil {
		return
	}
	queryName, queryType := queryNameFromSQL(data.SQL)
	queryData.queryName = queryName

	queryData.span.SetAttributes(
		SQLCQueryNameKey.String(queryName),
		SQLCQueryTypeKey.String(queryType),
	)
	queryData.span.SetName(queryName)
	if dt.includeQueryText {
		queryData.span.SetAttributes(semconv.DBQueryText(data.SQL))
	}

	var logAttrs []slog.Attr

	if data.Err != nil {
		queryData.span.SetStatus(codes.Error, data.Err.Error())
		queryData.span.RecordError(data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
	} else {
		queryData.span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.String("commandTag", data.CommandTag.String()))
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.String("sql", data.SQL),
			slog.Any("args", dt.logQueryArgs(data.Args)),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)
		dt.logger.LogAttrs(ctx, slog.LevelError,
			queryName,
			logAttrs...,
		)
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	if queryData == nil {
		return
	}
	defer queryData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	dt.recordHistogramMetric(ctx, "batch", queryData.queryName, interval, data.Err)

	var logAttrs []slog.Attr

	if data.Err != nil {
		dt.recordSpanError(queryData.span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
	} else {

		queryData.span.SetStatus(codes.Ok, "")
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.Duration("interval", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)
		dt.logger.LogAttrs(ctx, slog.LevelError,
			"batch queries",
			logAttrs...,
		)
	}
}
