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

type tracePrepareData struct {
	span          trace.Span // 16 bytes
	startTime     time.Time  // 16 bytes
	queryName     string     // 16 bytes
	queryType     string
	sql           string // 16 bytes
	statementName string // 16 bytes
}

func (dt *dbTracer) TracePrepareStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	queryName, queryType := queryNameFromSQL(data.SQL)
	ctx, span := dt.startSpan(ctx, "postgresql.prepare")
	span.SetAttributes(
		PGXOperationTypeKey.String("prepare"),
		PGXPrepareStmtNameKey.String(data.Name),
		SQLCQueryNameKey.String(queryName),
		SQLCQueryTypeKey.String(queryType),
	)

	if dt.includeQueryText {
		span.SetAttributes(semconv.DBQueryText(data.SQL))
	}

	return context.WithValue(ctx, dbTracerPrepareCtxKey, &tracePrepareData{
		startTime:     time.Now(),
		statementName: data.Name,
		span:          span,
		sql:           data.SQL,
		queryName:     queryName,
		queryType:     queryType,
	})
}

func (dt *dbTracer) TracePrepareEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareEndData,
) {
	prepareData := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)
	defer prepareData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)
	dt.recordHistogramMetric(ctx, "prepare", prepareData.queryName, interval, data.Err)

	var logAttrs []slog.Attr
	var level slog.Level

	if data.Err != nil {
		dt.recordSpanError(prepareData.span, data.Err)
		logAttrs = append(logAttrs, slog.String("error", data.Err.Error()))
		level = slog.LevelError
	} else {
		prepareData.span.SetStatus(codes.Ok, "")
		logAttrs = append(logAttrs, slog.Bool("alreadyPrepared", data.AlreadyPrepared))
		level = slog.LevelInfo
	}

	if dt.shouldLog(data.Err) {
		logAttrs = append(logAttrs, slog.String("statement_name", prepareData.statementName),
			slog.String("sql", prepareData.sql),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
		)

		dt.logger.LogAttrs(ctx, level,
			"prepare",
			logAttrs...,
		)
	}
}
