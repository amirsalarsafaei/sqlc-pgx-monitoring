package db

import (
	"context"
	"fmt"

	"github.com/amirsalarsafaei/sqlc-pgx-metrics/dbtracer"
	"github.com/amirsalarsafaei/sqlc-pgx-metrics/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type DBConfig struct {
	User string
	Pwd  string
	Host string
	Port string
	Name string
}

func GetConnectionPool(ctx context.Context, dbConf DBConfig, level logger.LogLevel) (*pgxpool.Pool, error) {
	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbConf.User, dbConf.Pwd, dbConf.Host, dbConf.Port, dbConf.Name,
	)
	poolConfig, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %v", err.Error())
	}

	poolConfig.ConnConfig.Tracer = dbtracer.NewDBTracer(
		logger.NewLogger(logrus.New()),
		level,
		prometheus.DefaultRegisterer,
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	return pool, pool.Ping(ctx)
}
