package poolstatus

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	// instrumentationName is the name of the instrumentation library.
	instrumentationName = "github.com/amirsalarsafaei/sqlc-pgx-monitoring"
	// stateKey is an attribute key for the state of a connection (e.g., "used", "idle").
	stateKey = attribute.Key("state")
	// reasonKey is an attribute key for the reason a connection was closed (e.g., "lifetime", "idletime").
	reasonKey = attribute.Key("reason")
)

var (
	// stateUsed is an attribute for a connection that is currently in use.
	stateUsed = stateKey.String("used")
	// stateIdle is an attribute for a connection that is currently idle.
	stateIdle = stateKey.String("idle")
	// reasonLifetime is an attribute for a connection closed due to exceeding its maximum lifetime.
	reasonLifetime = reasonKey.String("lifetime")
	// reasonIdleTime is an attribute for a connection closed due to exceeding its maximum idle time.
	reasonIdleTime = reasonKey.String("idletime")
)

// Stater is an interface that provides access to pgxpool statistics.
// It is implemented by *pgxpool.Pool.
type Stater interface {
	Stat() *pgxpool.Stat
}

// config holds the configuration for the metrics registration.
type config struct {
	meter      metric.Meter
	attributes []attribute.KeyValue
}

// Option is a function that configures the metrics registration.
type Option func(*config)

// WithMeterProvider sets the OpenTelemetry MeterProvider to be used for metrics.
// If not provided, the global MeterProvider will be used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return func(c *config) {
		c.meter = provider.Meter(
			instrumentationName,
		)
	}
}

// WithAttributes adds a set of attributes to all registered metrics.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return func(c *config) {
		c.attributes = attrs
	}
}

// Register registers the pgxpool metrics with OpenTelemetry.
// It takes a Stater (like *pgxpool.Pool) and optional configuration.
// It returns an error if any of the metrics fail to register.
func Register(stater Stater, opts ...Option) error {
	cfg := &config{
		meter: otel.GetMeterProvider().Meter(
			instrumentationName,
		),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	usage, err := cfg.meter.Int64ObservableGauge(
		semconv.DBClientConnectionsUsageName,
		metric.WithDescription(semconv.DBClientConnectionsUsageDescription),
		metric.WithUnit(semconv.DBClientConnectionsUsageUnit),
	)
	if err != nil {
		return fmt.Errorf("failed to create usage metric: %w", err)
	}

	maxConns, err := cfg.meter.Int64ObservableGauge(
		semconv.DBClientConnectionMaxName,
		metric.WithDescription(semconv.DBClientConnectionMaxDescription),
		metric.WithUnit(semconv.DBClientConnectionMaxUnit),
	)
	if err != nil {
		return fmt.Errorf("failed to create max connections metric: %w", err)
	}

	pending, err := cfg.meter.Int64ObservableGauge(
		semconv.DBClientConnectionsPendingRequestsName,
		metric.WithDescription(semconv.DBClientConnectionsPendingRequestsDescription),
		metric.WithUnit(semconv.DBClientConnectionPendingRequestsUnit),
	)
	if err != nil {
		return fmt.Errorf("failed to create pending requests metric: %w", err)
	}

	acquireCount, err := cfg.meter.Int64ObservableCounter(
		"pgx.pool.acquires",
		metric.WithDescription("Cumulative count of successful acquires from the pool."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create acquire count metric: %w", err)
	}

	canceledAcquireCount, err := cfg.meter.Int64ObservableCounter(
		"pgx.pool.canceled_acquires",
		metric.WithDescription("Cumulative count of acquires from the pool that were canceled by a context."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create canceled acquire count metric: %w", err)
	}

	waitedForAcquireCount, err := cfg.meter.Int64ObservableCounter(
		"pgx.pool.waited_for_acquires",
		metric.WithDescription("Cumulative count of acquires that waited for a resource to be released or constructed because the pool was empty."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create waited for acquire count metric: %w", err)
	}

	connsCreated, err := cfg.meter.Int64ObservableCounter(
		"pgx.pool.connections.created",
		metric.WithDescription("Cumulative count of new connections opened."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create connections created metric: %w", err)
	}

	connsDestroyed, err := cfg.meter.Int64ObservableCounter(
		"pgx.pool.connections.destroyed",
		metric.WithDescription("Cumulative count of connections destroyed, with a reason attribute."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create connections destroyed metric: %w", err)
	}

	acquireDuration, err := cfg.meter.Float64ObservableCounter(
		"pgx.pool.acquire.duration",
		metric.WithDescription("Total duration of all successful acquires from the pool."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create acquire duration metric: %w", err)
	}

	// This is the newly added metric.
	waitedForAcquireDuration, err := cfg.meter.Float64ObservableCounter(
		"pgx.pool.acquire.wait.duration",
		metric.WithDescription("The cumulative time successful acquires from the pool waited for a resource to be released or constructed because the pool was empty."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create waited for acquire duration metric: %w", err)
	}

	_, err = cfg.meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			stats := stater.Stat()
			obsOpts := metric.WithAttributes(cfg.attributes...)

			// Gauges
			o.ObserveInt64(usage, int64(stats.AcquiredConns()), metric.WithAttributes(stateUsed), obsOpts)
			o.ObserveInt64(usage, int64(stats.IdleConns()), metric.WithAttributes(stateIdle), obsOpts)
			o.ObserveInt64(maxConns, int64(stats.MaxConns()), obsOpts)
			o.ObserveInt64(pending, int64(stats.ConstructingConns()), obsOpts)

			// Counters
			o.ObserveInt64(acquireCount, stats.AcquireCount(), obsOpts)
			o.ObserveInt64(canceledAcquireCount, stats.CanceledAcquireCount(), obsOpts)
			o.ObserveInt64(waitedForAcquireCount, stats.EmptyAcquireCount(), obsOpts)
			o.ObserveInt64(connsCreated, stats.NewConnsCount(), obsOpts)

			o.ObserveInt64(connsDestroyed, stats.MaxLifetimeDestroyCount(), metric.WithAttributes(reasonLifetime), obsOpts)
			o.ObserveInt64(connsDestroyed, stats.MaxIdleDestroyCount(), metric.WithAttributes(reasonIdleTime), obsOpts)

			// Duration Counters
			o.ObserveFloat64(acquireDuration, stats.AcquireDuration().Seconds(), obsOpts)
			// Observe the newly added metric.
			o.ObserveFloat64(waitedForAcquireDuration, stats.EmptyAcquireWaitTime().Seconds(), obsOpts)

			return nil
		},
		// Register all instruments with the callback.
		usage, maxConns, pending, acquireCount, canceledAcquireCount, waitedForAcquireCount,
		connsCreated, connsDestroyed, acquireDuration, waitedForAcquireDuration,
	)
	if err != nil {
		return fmt.Errorf("failed to register metric callback: %w", err)
	}

	return nil
}
