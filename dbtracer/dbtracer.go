package dbtracer

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/pkg/prometheustools"
)

// used code from https://github.com/jackc/pgx-logrus

// dbTracer implements pgx.QueryTracer, pgx.BatchTracer, pgx.ConnectTracer, and pgx.CopyFromTracer
type dbTracer struct {
	logger      *logrus.Logger
	queryTiming prometheustools.Observer
	shouldLog   ShouldLog
}

func NewDBTracer(logger *logrus.Logger, registerer prometheus.Registerer,
	opts ...Option,
) pgx.QueryTracer {
	optCtx := optionCtx{
		name:    "sqlc_query_timing",
		help:    "database query timings by sqlc query name and status",
		buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.050, 0.100, 0.150, 0.200, 0.300, 0.500, 0.750, 1.0},
		shouldLog: func(_ error) bool {
			return true
		},
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
		queryTiming: queryTiming,
	}
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
	startTime time.Time
	sql       string
	args      []any
	queryName string
}

func (dt *dbTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, dbTracerQueryCtxKey, &traceQueryData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
		queryName: queryNameFromSQL(data.SQL),
	})
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	queryData := ctx.Value(dbTracerQueryCtxKey).(*traceQueryData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	labels := map[string]string{
		"query_name": queryData.queryName,
		"status":     "success",
	}
	defer func(labels map[string]string, interval time.Duration) {
		dt.queryTiming.With(labels).Observe(interval.Seconds())
	}(labels, interval)

	log := dt.logger.WithContext(ctx).WithFields(
		logrus.Fields{
			"sql":  queryData.sql,
			"args": logQueryArgs(queryData.args),
			"time": interval,
			"pid":  extractConnectionID(conn),
		},
	)

	if data.Err != nil {
		labels["status"] = "error"

		if dt.shouldLog(data.Err) {
			log.Errorf("Query: %s", queryData.queryName)
		}
	} else {
		log.WithField("commandTag", data.CommandTag.String()).Infof("Query: %s", queryData.queryName)
	}
}

type traceBatchData struct {
	startTime time.Time
	queryName string
}

func (dt *dbTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	return context.WithValue(ctx, dbTracerBatchCtxKey, &traceBatchData{
		startTime: time.Now(),
	})
}

func (dt *dbTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)
	queryData.queryName = queryNameFromSQL(data.SQL)

	log := dt.logger.WithContext(ctx).WithFields(
		logrus.Fields{
			"sql":  data.SQL,
			"args": logQueryArgs(data.Args),
			"pid":  extractConnectionID(conn),
		},
	)

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			log.Error("Query")
		}
	} else {
		log.WithField("commandTag", data.CommandTag.String()).Info("Query")
	}
}

func (dt *dbTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	queryData := ctx.Value(dbTracerBatchCtxKey).(*traceBatchData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	labels := map[string]string{
		"query_name": queryData.queryName,
		"status":     "success",
	}
	defer func(labels map[string]string, interval time.Duration) {
		dt.queryTiming.With(labels).Observe(interval.Seconds())
	}(labels, interval)

	log := dt.logger.WithContext(ctx).WithFields(
		logrus.Fields{
			"interval": interval,
			"pid":      extractConnectionID(conn),
		},
	)

	if data.Err != nil {
		labels["status"] = "error"
		if dt.shouldLog(data.Err) {
			log.Errorf("Query: %s", queryData.queryName)
		}
	}

	log.Info("Query")
}

type traceCopyFromData struct {
	startTime   time.Time
	TableName   pgx.Identifier
	ColumnNames []string
}

func (dt *dbTracer) TraceCopyFromStart(ctx context.Context,
	_ *pgx.Conn, data pgx.TraceCopyFromStartData,
) context.Context {
	return context.WithValue(ctx, dbTracerCopyFromCtxKey, &traceCopyFromData{
		startTime:   time.Now(),
		TableName:   data.TableName,
		ColumnNames: data.ColumnNames,
	})
}

func (dt *dbTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	copyFromData := ctx.Value(dbTracerCopyFromCtxKey).(*traceCopyFromData)

	endTime := time.Now()
	interval := endTime.Sub(copyFromData.startTime)

	log := dt.logger.WithContext(ctx).WithFields(
		logrus.Fields{
			"tableName":   copyFromData.TableName,
			"columnNames": copyFromData.ColumnNames,
			"time":        interval,
			"pid":         extractConnectionID(conn),
		},
	)

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			log.WithError(data.Err).Error("CopyFrom")
		}
	} else {
		log.WithField("rowCount", data.CommandTag.RowsAffected()).Info("CopyFrom")
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

	log := dt.logger.WithContext(ctx).
		WithFields(logrus.Fields{
			"host":     connectData.connConfig.Host,
			"port":     connectData.connConfig.Port,
			"database": connectData.connConfig.Database,
			"time":     interval,
		},
		)

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			log.WithError(data.Err).Error("database connect")
		}
		return
	}

	if data.Conn != nil {
		log.Info("database connect")
	}
}

type tracePrepareData struct {
	startTime time.Time
	name      string
	sql       string
}

func (dt *dbTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	return context.WithValue(ctx, dbTracerPrepareCtxKey, &tracePrepareData{
		startTime: time.Now(),
		name:      data.Name,
		sql:       data.SQL,
	})
}

func (dt *dbTracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	prepareData := ctx.Value(dbTracerPrepareCtxKey).(*tracePrepareData)

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)

	log := dt.logger.WithContext(ctx).
		WithFields(logrus.Fields{
			"name": prepareData.name,
			"sql":  prepareData.sql,
			"time": interval,
			"pid":  extractConnectionID(conn),
		},
		)

	if data.Err != nil {
		if dt.shouldLog(data.Err) {
			log.WithError(data.Err).Error("Prepare")
		}
	} else {
		log.WithField("alreadyPrepared", data.AlreadyPrepared).Info("Prepare")
	}
}

func logQueryArgs(args []any) []any {
	logArgs := make([]any, 0, len(args))

	for _, a := range args {
		switch v := a.(type) {
		case []byte:
			if len(v) < 64 {
				a = hex.EncodeToString(v)
			} else {
				a = fmt.Sprintf("%x (truncated %d bytes)", v[:64], len(v)-64)
			}
		case string:
			if len(v) > 64 {
				var l int
				for w := 0; l < 64; l += w {
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
