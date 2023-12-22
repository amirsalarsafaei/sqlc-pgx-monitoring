package dbtracer

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/pkg/logger"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/pkg/prometheustools"
)

// used code from https://github.com/jackc/pgx-logrus

// dbTracer implements pgx.QueryTracer, pgx.BatchTracer, pgx.ConnectTracer, and pgx.CopyFromTracer
type dbTracer struct {
	logger   logger.Logger
	logLevel logger.LogLevel

	queryTiming prometheustools.Observer
}

func NewDBTracer(logger logger.Logger, logLevel logger.LogLevel, registerer prometheus.Registerer,
	opts ...Option) pgx.QueryTracer {

	optCtx := optionCtx{
		name:    "sqlc_query_timing",
		help:    "sqlc query timings",
		buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.050, 0.100, 0.150, 0.200, 0.300, 0.500, 0.750, 1.0},
	}
	for _, opt := range opts {
		opt(&optCtx)
	}

	queryTiming := prometheustools.NewHistogram(
		optCtx.name,
		optCtx.help,
		optCtx.buckets,
		registerer,
		"query_name", "status")

	return &dbTracer{
		logger:      logger,
		logLevel:    logLevel,
		queryTiming: queryTiming,
	}
}

type ctxKey int

const (
	_ ctxKey = iota
	tracelogQueryCtxKey
	tracelogBatchCtxKey
	tracelogCopyFromCtxKey
	tracelogConnectCtxKey
	tracelogPrepareCtxKey
)

type traceQueryData struct {
	startTime time.Time
	sql       string
	args      []any
	queryName string
}

func (dt *dbTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, tracelogQueryCtxKey, &traceQueryData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
		queryName: queryNameFromSQL(data.SQL),
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	queryData := ctx.Value(tracelogQueryCtxKey).(*traceQueryData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	labels := map[string]string{
		"query_name": queryData.queryName,
		"status":     "success",
	}
	defer dt.queryTiming.With(labels).Observe(interval.Seconds())

	if data.Err != nil {
		labels["status"] = "error"
		if dt.shouldLog(logger.LogLevelError) {
			dt.log(ctx, conn, logger.LogLevelError, "Query", map[string]any{"sql": queryData.sql,
				"args": logger.LogQueryArgs(queryData.args), "err": data.Err, "time": interval})
		}
		return
	}

	if dt.shouldLog(logger.LogLevelInfo) {
		dt.log(ctx, conn, logger.LogLevelInfo, "Query", map[string]any{"sql": queryData.sql,
			"args": logger.LogQueryArgs(queryData.args), "time": interval, "commandTag": data.CommandTag.String()})
	}
}

type traceBatchData struct {
	startTime time.Time
	queryName string
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	return context.WithValue(ctx, tracelogBatchCtxKey, &traceBatchData{
		startTime: time.Now(),
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	queryData := ctx.Value(tracelogBatchCtxKey).(*traceBatchData)
	queryData.queryName = queryNameFromSQL(data.SQL)

	if data.Err != nil {
		if dt.shouldLog(logger.LogLevelError) {
			dt.log(ctx, conn, logger.LogLevelError, "BatchQuery", map[string]any{"sql": data.SQL,
				"args": logger.LogQueryArgs(data.Args), "err": data.Err})
		}
		return
	}

	if dt.shouldLog(logger.LogLevelInfo) {
		dt.log(ctx, conn, logger.LogLevelInfo, "BatchQuery", map[string]any{"sql": data.SQL,
			"args": logger.LogQueryArgs(data.Args), "commandTag": data.CommandTag.String()})
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	queryData := ctx.Value(tracelogBatchCtxKey).(*traceBatchData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	if data.Err != nil {
		if dt.shouldLog(logger.LogLevelError) {
			dt.log(ctx, conn, logger.LogLevelError,
				"BatchClose", map[string]any{"err": data.Err, "time": interval})
		}
		return
	}

	if dt.shouldLog(logger.LogLevelInfo) {
		dt.log(ctx, conn, logger.LogLevelInfo, "BatchClose", map[string]any{"time": interval})
	}
}

type traceCopyFromData struct {
	startTime   time.Time
	TableName   pgx.Identifier
	ColumnNames []string
}

func (dt *dbTracer) TraceCopyFromStart(ctx context.Context,
	_ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	return context.WithValue(ctx, tracelogCopyFromCtxKey, &traceCopyFromData{
		startTime:   time.Now(),
		TableName:   data.TableName,
		ColumnNames: data.ColumnNames,
	})
}

func (dt *dbTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	copyFromData := ctx.Value(tracelogCopyFromCtxKey).(*traceCopyFromData)

	endTime := time.Now()
	interval := endTime.Sub(copyFromData.startTime)

	labels := map[string]string{
		"query_name": fmt.Sprintf("copyFrom_%s", copyFromData.TableName),
		"status":     "success",
	}
	defer dt.queryTiming.With(labels).Observe(interval.Seconds())

	if data.Err != nil {
		labels["status"] = "error"
		if dt.shouldLog(logger.LogLevelError) {
			dt.log(ctx, conn, logger.LogLevelError, "CopyFrom", map[string]any{"tableName": copyFromData.TableName,
				"columnNames": copyFromData.ColumnNames, "err": data.Err, "time": interval})
		}
		return
	}

	if dt.shouldLog(logger.LogLevelInfo) {
		dt.log(ctx, conn, logger.LogLevelInfo, "CopyFrom", map[string]any{"tableName": copyFromData.TableName,
			"columnNames": copyFromData.ColumnNames, "err": data.Err, "time": interval,
			"rowCount": data.CommandTag.RowsAffected()})
	}
}

type traceConnectData struct {
	startTime  time.Time
	connConfig *pgx.ConnConfig
}

func (dt *dbTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	return context.WithValue(ctx, tracelogConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (dt *dbTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	connectData := ctx.Value(tracelogConnectCtxKey).(*traceConnectData)

	endTime := time.Now()
	interval := endTime.Sub(connectData.startTime)

	if data.Err != nil {
		if dt.shouldLog(logger.LogLevelError) {
			dt.logger.Log(ctx, logger.LogLevelError, "Connect", map[string]any{
				"host":     connectData.connConfig.Host,
				"port":     connectData.connConfig.Port,
				"database": connectData.connConfig.Database,
				"time":     interval,
				"err":      data.Err,
			})
		}
		return
	}

	if data.Conn != nil {
		if dt.shouldLog(logger.LogLevelInfo) {
			dt.log(ctx, data.Conn, logger.LogLevelInfo, "Connect", map[string]any{
				"host":     connectData.connConfig.Host,
				"port":     connectData.connConfig.Port,
				"database": connectData.connConfig.Database,
				"time":     interval,
			})
		}
	}
}

type tracePrepareData struct {
	startTime time.Time
	name      string
	sql       string
}

func (dt *dbTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn,
	data pgx.TracePrepareStartData) context.Context {
	return context.WithValue(ctx, tracelogPrepareCtxKey, &tracePrepareData{
		startTime: time.Now(),
		name:      data.Name,
		sql:       data.SQL,
	})
}

func (dt *dbTracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	prepareData := ctx.Value(tracelogPrepareCtxKey).(*tracePrepareData)

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)

	if data.Err != nil {
		if dt.shouldLog(logger.LogLevelError) {
			dt.log(ctx, conn, logger.LogLevelError, "Prepare", map[string]any{"name": prepareData.name,
				"sql": prepareData.sql, "err": data.Err, "time": interval})
		}
		return
	}

	if dt.shouldLog(logger.LogLevelInfo) {
		dt.log(ctx, conn, logger.LogLevelInfo, "Prepare", map[string]any{"name": prepareData.name,
			"sql": prepareData.sql, "time": interval, "alreadyPrepared": data.AlreadyPrepared})
	}
}

func (dt *dbTracer) shouldLog(lvl logger.LogLevel) bool {
	return dt.logLevel >= lvl
}

func (dt *dbTracer) log(ctx context.Context, conn *pgx.Conn, lvl logger.LogLevel, msg string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}

	pgConn := conn.PgConn()
	if pgConn != nil {
		pid := pgConn.PID()
		if pid != 0 {
			data["pid"] = pid
		}
	}

	dt.logger.Log(ctx, lvl, msg, data)
}
