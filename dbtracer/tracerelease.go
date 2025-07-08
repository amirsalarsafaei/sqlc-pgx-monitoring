package dbtracer

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
)

var pgxPoolConnOperationReleased = PGXPoolConnOperationKey.String("release")

// TraceRelease implements Tracer.
func (dt *dbTracer) TraceRelease(pool *pgxpool.Pool, data pgxpool.TraceReleaseData) {
	dt.connReleaseCounter.Add(context.Background(), 1, metric.WithAttributes(
		pgxPoolConnOperationReleased,
	))
}
