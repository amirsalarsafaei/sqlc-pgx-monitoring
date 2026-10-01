package dbtracer

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type traceCopyFromData struct {
	span        trace.Span
	columnNames []string
	startTime   time.Time
	tableName   pgx.Identifier
}

var pgxOperationCopyFrom = PGXOperationTypeKey.String("copy_from")

func (dt *dbTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	ctx, span := dt.startSpan(ctx, "postgresql.copy_from",
		pgxOperationCopyFrom,
		semconv.DBCollectionName(data.TableName.Sanitize()),
	)

	return context.WithValue(ctx, dbTracerCopyFromCtxKey, &traceCopyFromData{
		span:        span,
		startTime:   time.Now(),
		tableName:   data.TableName,
		columnNames: data.ColumnNames,
	})
}

func (dt *dbTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	traceData, ok := ctx.Value(dbTracerCopyFromCtxKey).(*traceCopyFromData)
	if !ok || traceData == nil {
		return
	}

	interval := time.Since(traceData.startTime)
	dt.recordDBOperationHistogramMetric(ctx, "copy_from", nil, interval, data.Err)
	endSpan(traceData.span, data.Err)

	if !dt.shouldLog(data.Err) {
		return
	}

	var logAttrs []slog.Attr
	if data.Err == nil {
		logAttrs = append(logAttrs, slog.Int64("rowCount", data.CommandTag.RowsAffected()))
	}
	logAttrs = append(logAttrs,
		slog.Any("tableName", traceData.tableName),
		slog.Any("columnNames", traceData.columnNames),
		slog.Duration("time", interval),
		slog.Uint64("pid", uint64(extractConnectionID(conn))),
	)

	dt.log(ctx, "copy_from", data.Err, logAttrs...)
}
