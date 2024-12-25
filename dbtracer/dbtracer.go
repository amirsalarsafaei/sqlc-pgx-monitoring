package dbtracer

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	pgx.BatchTracer
	pgx.ConnectTracer
	pgx.CopyFromTracer
	pgx.QueryTracer
	pgx.PrepareTracer
}

// dbTracer implements pgx.QueryTracer, pgx.BatchTracer, pgx.ConnectTracer, and pgx.CopyFromTracer
type dbTracer struct {
	logger          *slog.Logger
	tracer          trace.Tracer
	shouldLog       ShouldLog
	databaseName    string
	logArgs         bool
	logArgsLenLimit int
	histogram       metric.Float64Histogram
}

func NewDBTracer(
	databaseName string,
	opts ...Option,
) (Tracer, error) {
	optCtx := optionCtx{
		name: "github.com/amirsalarsafaei/sqlc-pgx-monitoring",
		shouldLog: func(_ error) bool {
			return true
		},
		meterProvider:   otel.GetMeterProvider(),
		traceProvider:   otel.GetTracerProvider(),
		logArgs:         true,
		logArgsLenLimit: 64,
		latencyHistogramConfig: struct {
			name        string
			unit        string
			description string
		}{
			description: "The duration of database queries by sqlc function names",
			unit:        "s",
			name:        "db_query_duration",
		},
	}
	for _, opt := range opts {
		opt(&optCtx)
	}

	meter := optCtx.meterProvider.Meter(optCtx.name)
	histogram, err := meter.Float64Histogram(
		optCtx.latencyHistogramConfig.name,
		metric.WithDescription(optCtx.latencyHistogramConfig.description),
		metric.WithUnit(optCtx.latencyHistogramConfig.unit),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing histogram meter: [%w]", err)
	}

	return &dbTracer{
		logger:       slog.Default(),
		databaseName: databaseName,
		tracer:       optCtx.traceProvider.Tracer(optCtx.name),
		shouldLog:    optCtx.shouldLog,
		logArgs:      optCtx.logArgs,
		histogram:    histogram,
	}, nil
}

type ctxKey int

const (
	_ ctxKey = iota
	dbTracerQueryCtxKey
	dbTracerBatchCtxKey
	dbTracerCopyFromCtxKey
	dbTracerConnectCtxKey
	dbTracerPrepareCtxKey
)

type traceQueryData struct {
	args      []any      // 24 bytes
	span      trace.Span // 16 bytes
	sql       string     // 16 bytes
	queryName string     // 16 bytes
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
		span:      span,
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "query"),
		attribute.String("query_name", queryData.queryName),
	))

	defer queryData.span.End()

	if data.Err != nil {
		queryData.span.SetStatus(codes.Error, data.Err.Error())
		queryData.span.RecordError(data.Err)

		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				fmt.Sprintf("Query: %s", queryData.queryName),
				slog.String("sql", queryData.sql),
				slog.Any("args", dt.logQueryArgs(queryData.args)),
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
			slog.Any("args", dt.logQueryArgs(queryData.args)),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.String("commandTag", data.CommandTag.String()),
		)
	}
}

type traceBatchData struct {
	span      trace.Span // 16 bytes
	startTime time.Time  // 16 bytes
	queryName string     // 16 bytes
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	ctx, span := dt.tracer.Start(ctx, "postgresql.batch")
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

type traceConnectData struct {
	startTime  time.Time
	connConfig *pgx.ConnConfig
}

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	return context.WithValue(ctx, dbTracerConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	connectData := ctx.Value(dbTracerConnectCtxKey).(*traceConnectData)

	endTime := time.Now()
	interval := endTime.Sub(connectData.startTime)

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"database connect",
				slog.String("host", connectData.connConfig.Host),
				slog.Uint64("port", uint64(connectData.connConfig.Port)),
				slog.String("database", connectData.connConfig.Database),
				slog.Duration("time", interval),
				slog.String("error", data.Err.Error()),
			)
		}
		return

	}

	dt.logger.LogAttrs(ctx, slog.LevelInfo,
		"database connect",
		slog.String("host", connectData.connConfig.Host),
		slog.Uint64("port", uint64(connectData.connConfig.Port)),
		slog.String("database", connectData.connConfig.Database),
		slog.Duration("time", interval),
	)

	if data.Conn != nil {
		dt.logger.LogAttrs(ctx, slog.LevelInfo,
			"database connect",
			slog.String("host", connectData.connConfig.Host),
			slog.Uint64("port", uint64(connectData.connConfig.Port)),
			slog.String("database", connectData.connConfig.Database),
			slog.Duration("time", interval),
		)
	}
}

type tracePrepareData struct {
	span      trace.Span // 16 bytes
	startTime time.Time  // 16 bytes
	name      string     // 16 bytes
	sql       string     // 16 bytes
}

func (dt *dbTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	ctx, span := dt.tracer.Start(ctx, "postgresql.prepare")
	span.SetAttributes(
		attribute.String("db.name", dt.databaseName),
		attribute.String("db.operation", "prepare"),
		attribute.String("db.prepared_statement_name", data.Name),
	)
	return context.WithValue(ctx, dbTracerPrepareCtxKey, &tracePrepareData{
		startTime: time.Now(),
		name:      data.Name,
		span:      span,
		sql:       data.SQL,
	})
}

func (dt *dbTracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	prepareData := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)
	defer prepareData.span.End()

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)
	dt.histogram.Record(ctx, interval.Seconds(), metric.WithAttributes(
		attribute.String("operation", "prepare"),
		attribute.String("statement_name", prepareData.name),
	))

	if data.Err != nil {
		prepareData.span.SetStatus(codes.Error, data.Err.Error())
		prepareData.span.RecordError(data.Err)
		if dt.shouldLog(data.Err) {
			dt.logger.LogAttrs(ctx, slog.LevelError,
				"Prepare",
				slog.String("name", prepareData.name),
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
			slog.String("name", prepareData.name),
			slog.String("sql", prepareData.sql),
			slog.Duration("time", interval),
			slog.Uint64("pid", uint64(extractConnectionID(conn))),
			slog.Bool("alreadyPrepared", data.AlreadyPrepared),
		)
	}
}

func (dt *dbTracer) logQueryArgs(args []any) []any {
	if !dt.logArgs {
		return nil
	}

	logArgs := make([]any, 0, len(args))
	limit := dt.logArgsLenLimit
	if limit == 0 {
		limit = 64 // default limit if not set
	}

	for _, a := range args {
		switch v := a.(type) {
		case []byte:
			if len(v) < limit {
				a = hex.EncodeToString(v)
			} else {
				a = fmt.Sprintf("%x (truncated %d bytes)", v[:limit], len(v)-limit)
			}
		case string:
			if len(v) > limit {
				var l int
				for w := 0; l < limit; l += w {
					_, w = utf8.DecodeRuneInString(v[l:])
				}

				if len(v) > l {
					a = fmt.Sprintf("%s (truncated %d bytes)", v[:l], len(v)-l)
				}
			}
		}

		logArgs = append(logArgs, a)
	}

	return logArgs
}

func extractConnectionID(conn *pgx.Conn) uint32 {
	if conn == nil {
		return 0
	}

	pgConn := conn.PgConn()
	if pgConn != nil {
		pid := pgConn.PID()
		return pid
	}
	return 0
}
