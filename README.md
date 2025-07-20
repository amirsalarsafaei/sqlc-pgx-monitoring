# sqlc-pgx-monitoring

![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)
[![Latest Release](https://img.shields.io/github/v/release/amirsalarsafaei/sqlc-pgx-monitoring)](https://github.com/amirsalarsafaei/sqlc-pgx-monitoring/releases/latest)
[![codecov](https://codecov.io/github/amirsalarsafaei/sqlc-pgx-monitoring/graph/badge.svg?token=NT5PXGLJMS)](https://codecov.io/github/amirsalarsafaei/sqlc-pgx-monitoring)

`sqlc-pgx-monitoring` is a Go package that offers powerful query time monitoring and logging capabilities for applications using the popular `pgx` and `sqlc` libraries in Golang. If you want to gain insights into the performance of your PostgreSQL database queries and ensure the reliability of your application, this package is a valuable addition to your toolset.

## Features

- **Complete OpenTelemetry Support**: Built-in integration with OpenTelemetry for comprehensive observability, including metrics, traces, and spans for all database operations. Traces every database interaction including:

  - Individual queries
  - Batch operations
  - Prepared statements
  - Connection lifecycle
  - COPY FROM operations
  - Pool connection acquire/release operations

- **OpenTelemetry Semantic Convention**: Aligned with Otel's semantic convetions ([defined here](https://opentelemetry.io/docs/specs/semconv/database/)).

- **Pool Status Monitoring**: Comprehensive monitoring of pgxpool connection pools with OpenTelemetry metrics including:

  - Active/idle connection counts
  - Maximum connections
  - Pending connection requests
  - Connection acquire/release counts and durations
  - Connection creation and destruction metrics
  - Wait times for pool resource acquisition

- **Pool Acquire Traces**: Detailed tracing of connection pool acquire operations with:

  - Individual acquire/release traces and spans
  - Connection acquisition timing and duration
  - Error tracking for failed acquisitions
  - Connection lifecycle visibility

- **Modern Structured Logging**: Native support for Go's `slog` package, providing structured, leveled logging that's easy to parse and analyze.

- **Query Time Monitoring**: Keep a close eye on the execution times of your SQL queries to identify and optimize slow or resource-intensive database operations. It uses name declared in sqlc queries in the label for distinguishing queries from each other.

- **Detailed Logging**: Record detailed logs of executed queries, including name, parameters, timings, and outcomes, which can be invaluable for debugging and performance analysis.

- **Compatible with `pgx` and `sqlc`**: Designed to seamlessly integrate with the `pgx` database driver and the `sqlc` code generation tool, making it a great fit for projects using these technologies.

## Installation

To get started with `sqlc-pgx-monitoring`, you can simply use `go get`:

```shell
go get github.com/amirsalarsafaei/sqlc-pgx-monitoring@latest
```

This will install the latest released version. You can also specify a particular version if needed:

```shell
go get github.com/amirsalarsafaei/sqlc-pgx-monitoring@v1.7.1
```

## Usage

To begin using `sqlc-pgx-monitoring` in your Go project, follow these basic steps:

1. Import the package:

   ```go
   import "github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
   ```

2. Before creating connection or connection pool, assign dbTracer in your connection config:

   ### pgx.Conn

   ```go
   connConfig.Tracer = dbtracer.NewDBTracer(
      "database_name",
   )
   ```

   ### pgxpool.Pool

   ```go
   poolConfig.ConnConfig.Tracer = dbtracer.NewDBTracer(
      "database_name",
   )
   ```

### Available Options

The `NewDBTracer` function accepts various options to customize its behavior:

#### Logging Options

- `WithLogger(logger *slog.Logger)`: Sets a custom structured logger for query logging
- `WithShouldLog(shouldLog ShouldLog)`: Configures when to log based on error conditions
- `WithLogArgs(enabled bool)`: Enables/disables logging of query arguments
- `WithLogArgsLenLimit(limit int)`: Sets maximum length for logged arguments

#### Telemetry Options

- `WithMeterProvider(mp metric.MeterProvider)`: Sets the OpenTelemetry meter provider for metrics
- `WithTraceProvider(tp trace.TracerProvider)`: Sets the OpenTelemetry tracer provider
- `WithLatencyHistogramConfig(name, unit, description string)`: Configures the latency histogram properties
  ```go
  dbtracer.NewDBTracer(
      "database_name",
      dbtracer.WithLatencyHistogramConfig(
          "custom_histogram_name",
          "ms",
          "Custom histogram description",
      ),
  )
  ```

#### Example Usage

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
mp := metric.NewMeterProvider()
tp := trace.NewTracerProvider()

tracer := dbtracer.NewDBTracer(
    "database_name",
    dbtracer.WithLogger(logger),
    dbtracer.WithMeterProvider(mp),
    dbtracer.WithTraceProvider(tp),
    dbtracer.WithLogArgs(true),
    dbtracer.WithLogArgsLenLimit(1000),
    dbtracer.WithShouldLog(func(err error) bool {
        return err != nil // Only log when there's an error
    }),
)
```

For more information refer to the [example](examples) directory, which includes Docker migration instructions and a complete monitoring setup.

### Pool Status Monitoring

To enable comprehensive pool monitoring with OpenTelemetry metrics, you need to register the pool status monitor:

```go
import (
    "github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"
)

// After creating your pgxpool
pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
if err != nil {
    return nil, err
}

// Register pool status monitoring
err = poolstatus.Register(pool)
if err != nil {
    log.Fatal("failed to register pool status monitoring:", err)
}
```

You can also customize the monitoring with additional options:

```go
import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// Register with custom meter provider and attributes
err = poolstatus.Register(pool,
    poolstatus.WithMeterProvider(customMeterProvider),
    poolstatus.WithAttributes(
        attribute.String("service.name", "my-service"),
        attribute.String("db.instance", "production"),
    ),
)
```

This will automatically expose the following OpenTelemetry metrics:

- `db.client.connections.usage` - Current connection usage (active/idle)
- `db.client.connections.max` - Maximum number of connections
- `db.client.connections.pending_requests` - Number of pending connection requests
- `pgx.pool.acquires` - Total successful connection acquisitions
- `pgx.pool.canceled_acquires` - Total canceled acquisitions
- `pgx.pool.waited_for_acquires` - Acquisitions that had to wait
- `pgx.pool.connections.created` - Total connections created
- `pgx.pool.connections.destroyed` - Total connections destroyed (with reason)
- `pgx.pool.acquire.duration` - Time spent acquiring connections
- `pgx.pool.acquire.wait.duration` - Time spent waiting for available connections

### Tracing and Monitoring Visualization

### Tracing

![Tracing](./docs/tracing-1.png)
![Tracing](./docs/tracing-2.png)

### Monitoring

![Monitoring](./docs/monitoring.png)

## License

`sqlc-pgx-monitoring` is open-source software licensed under the [MIT License](LICENSE). Feel free to use, modify, and distribute it according to the terms of this license.

## Contributing

We welcome contributions from the community. If you have suggestions, bug reports, or want to contribute to the development of `sqlc-pgx-monitoring`, please refer to our [contribution guidelines](CONTRIBUTING.md).

Happy querying and monitoring with `sqlc-pgx-monitoring`!
