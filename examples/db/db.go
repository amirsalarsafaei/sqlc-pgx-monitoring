package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
)

type DBConfig struct {
	User string
	Pwd  string
	Host string
	Port string
	Name string
}

func GetConnectionPool(ctx context.Context, dbConf DBConfig) (*pgxpool.Pool, error) {
	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbConf.User, dbConf.Pwd, dbConf.Host, dbConf.Port, dbConf.Name,
	)
	poolConfig, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %w", err)
	}

	tracer, err := dbtracer.NewDBTracer(
		"postgres",
	)
	if err != nil {
		return nil, fmt.Errorf("creating tracer: %w", err)
	}

	poolConfig.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	return pool, pool.Ping(ctx)
}

func GetConnection(ctx context.Context, dbConf DBConfig) (*pgx.Conn, error) {
	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbConf.User, dbConf.Pwd, dbConf.Host, dbConf.Port, dbConf.Name,
	)
	connConfig, err := pgx.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %v", err.Error())
	}

	connConfig.Tracer, err = dbtracer.NewDBTracer(
		"postgres",
	)
	if err != nil {
		return nil, fmt.Errorf("creating tracer: %w", err)
	}

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return nil, err
	}

	return conn, conn.Ping(ctx)
}
