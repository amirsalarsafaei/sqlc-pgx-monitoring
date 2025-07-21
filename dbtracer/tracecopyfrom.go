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

type traceCopyFromData struct {
	ColumnNames []string
	startTime   time.Time
	TableName   pgx.Identifier
}

var pgxOperationCopyFrom = PGXOperationTypeKey.String("copy_from")

func (dt *dbTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {

	ctx, _ = dt.getTracer().Start(ctx, "postgresql.copy_from", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			dt.infoAttrs...),
		trace.WithAttributes(
			pgxOperationCopyFrom,
			semconv.DBCollectionName(data.TableName.Sanitize())),
	)

	return context.WithValue(ctx, dbTracerCopyFromCtxKey, &traceCopyFromData{
		startTime:   time.Now(),
		TableName:   data.TableName,
		ColumnNames: data.ColumnNames,
	})
}

func (dt *dbTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	copyFromData := ctx.Value(dbTracerCopyFromCtxKey).(*traceCopyFromData)
	if copyFromData == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	defer span.End()

	interval := time.Since(copyFromData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "copy_from", nil, interval, data.Err)

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.Int64("rowCount", data.CommandTag.RowsAffected()))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.Any("tableName", copyFromData.TableName),
			slog.Any("columnNames", copyFromData.ColumnNames),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"copy_from",
			logAttrs...,
		)
	}
}
