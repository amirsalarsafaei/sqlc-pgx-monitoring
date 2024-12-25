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

type tracePrepareData struct {
	span          trace.Span // 16 bytes
	startTime     time.Time  // 16 bytes
	queryName     string     // 16 bytes
	queryType     string
	sql           string // 16 bytes
	statementName string // 16 bytes
}

func (dt *dbTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	queryName, queryType := queryNameFromSQL(data.SQL)
	ctx, span := dt.tracer.Start(ctx, "postgresql.prepare")
	span.SetAttributes(
		attribute.String("db.name", dt.databaseName),
		attribute.String("db.operation", "prepare"),
		attribute.String("db.prepared_statement_name", data.Name),
		attribute.String("db.query_name", queryName),
		attribute.String("db.query_type", queryType),
	)
	return context.WithValue(ctx, dbTracerPrepareCtxKey, &tracePrepareData{
		startTime:     time.Now(),
		statementName: data.Name,
		span:          span,
		sql:           data.SQL,
	})
}

func (dt *dbTracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	prepareData := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)
	defer prepareData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "prepare"),
		attribute.String("statement_name", prepareData.statementName),
		attribute.String("query_name", prepareData.queryName),
		attribute.String("query_type", prepareData.queryType),
		attribute.Bool("error", data.Err != nil),
	))

	if data.Err != nil {
		prepareData.span.SetStatus(codes.Error, data.Err.Error())
		prepareData.span.RecordError(data.Err)
		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"Prepare",
				slog.String("statement_name", prepareData.statementName),
				slog.String("sql", prepareData.sql),
				slog.Duration("time", interval),
				slog.Uint64("pid", uint64(extractConnectionID(conn))),
				slog.String("error", data.Err.Error()),
			)
		}
	} else {
		prepareData.span.SetStatus(codes.Ok, "")
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"Prepare",
			slog.String("statement_name", prepareData.statementName),
			slog.String("sql", prepareData.sql),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.Bool("alreadyPrepared", data.AlreadyPrepared),
		)
	}
}
