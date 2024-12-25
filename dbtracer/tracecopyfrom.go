package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type traceCopyFromData struct {
	ColumnNames []string       // 24 bytes
	span        trace.Span     // 16 bytes
	startTime   time.Time      // 16 bytes
	TableName   pgx.Identifier // slice - 24 bytes
}

func (dt *dbTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	ctx, span := dt.tracer.Start(ctx, "postgresql.copy_from")
	span.SetAttributes(
		attribute.String("db.name", dt.databaseName),
		attribute.String("db.operation", "copy"),
		attribute.String("db.table", data.TableName.Sanitize()),
	)
	return context.WithValue(ctx, dbTracerCopyFromCtxKey, &traceCopyFromData{
		startTime:   time.Now(),
		TableName:   data.TableName,
		ColumnNames: data.ColumnNames,
		span:        span,
	})
}

func (dt *dbTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	copyFromData := ctx.Value(dbTracerCopyFromCtxKey).(*traceCopyFromData)
	defer copyFromData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(copyFromData.startTime)
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "copy"),
		attribute.String("table", copyFromData.TableName.Sanitize()),
		attribute.Bool("error", data.Err != nil),
	))

	if data.Err != nil {
		copyFromData.span.SetStatus(codes.Error, data.Err.Error())
		copyFromData.span.RecordError(data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"CopyFrom",
				slog.Any("tableName", copyFromData.TableName),
				slog.Any("columnNames", copyFromData.ColumnNames),
				slog.Duration("time", interval),
				slog.Uint64("pid", uint64(extractConnectionID(conn))),
				slog.String("error", data.Err.Error()),
			)
		}
	} else {
		copyFromData.span.SetStatus(codes.Ok, "")
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"CopyFrom",
			slog.Any("tableName", copyFromData.TableName),
			slog.Any("columnNames", copyFromData.ColumnNames),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.Int64("rowCount", data.CommandTag.RowsAffected()),
		)
	}
}
