package db

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

func (c Config) ConnString() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, c.Port),
		Path:     c.Name,
		RawQuery: "sslmode=disable",
	}

	return u.String()
}

func NewPool(ctx context.Context, connString string, tracer dbtracer.Tracer, configure ...func(*pgxpool.Config)) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %w", err)
	}

	poolConfig.ConnConfig.Tracer = tracer
	for _, c := range configure {
		c(poolConfig)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	return pool, nil
}
