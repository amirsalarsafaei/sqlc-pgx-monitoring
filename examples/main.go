package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"examples/db"
	"examples/db/entities/exampletable"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"
)

// setupTelemetry initializes OpenTelemetry tracing and metrics
func setupTelemetry(ctx context.Context) (func(), error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(getEnv("OTEL_SERVICE_NAME", "sqlc-pgx-monitoring-example")),
			semconv.ServiceVersion(getEnv("OTEL_SERVICE_VERSION", "1.0.0")),
			semconv.DeploymentEnvironment("development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// Setup metrics
	// OTLP metrics exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Prometheus exporter for local metrics endpoint
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(5*time.Second))),
		metric.WithReader(promExporter),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: "*duration*"},
			metric.Stream{
				Aggregation: metric.AggregationExplicitBucketHistogram{
					Boundaries: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
				},
			},
		)),
	)
	otel.SetMeterProvider(meterProvider)

	// Set text map propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown tracer provider", "error", err)
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown meter provider", "error", err)
		}
	}, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// startMetricsServer starts the metrics HTTP server
func startMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 100 * time.Millisecond,
	}

	go func() {
		slog.Info("Starting metrics server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server error", "error", err)
		}
	}()

	return server
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func main() {
	ctx := context.Background()

	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting sqlc-pgx-monitoring example application")

	// Setup telemetry
	cleanup, err := setupTelemetry(ctx)
	if err != nil {
		slog.Error("Failed to setup telemetry", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// Start metrics server
	metricsServer := startMetricsServer()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown metrics server", "error", err)
		}
	}()

	// Setup database connection
	dbConfig := db.DBConfig{
		User: getEnv("DB_USER", "example"),
		Pwd:  getEnv("DB_PASSWORD", "complex-password"),
		Host: getEnv("DB_HOST", "localhost"),
		Port: getEnv("DB_PORT", "5432"),
		Name: getEnv("DB_NAME", "example_db"),
	}

	slog.Info("Connecting to database", "host", dbConfig.Host, "port", dbConfig.Port, "db", dbConfig.Name)

	pool, err := db.GetConnectionPool(ctx, dbConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	err = poolstatus.Register(pool)
	if err != nil {
		slog.Error("failed to create poolstatus monitor", "error", err)
		os.Exit(1)
	}

	slog.Info("Database connected successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create querier
	querier := exampletable.New()

	// Initial demonstration queries
	slog.Info("Running initial demonstration queries")

	exampleRows, err := querier.ExampleQuery(ctx, pool, pgtype.Text{
		String: "basdf",
		Valid:  true,
	})
	if err != nil {
		slog.Error("Failed to execute ExampleQuery", "error", err)
	} else {
		slog.Info("ExampleQuery completed", "rows_returned", len(exampleRows))
		for i, row := range exampleRows {
			slog.Info("Query result", "index", i, "foo", row.Foo.String)
		}
	}

	_, err = querier.ExampleQuery2(ctx, pool, pgtype.Text{
		String: "basdf",
		Valid:  true,
	})
	if err != nil {
		slog.Error("Failed to execute ExampleQuery2", "error", err)
	} else {
		slog.Info("ExampleQuery2 completed")
	}

	// Start continuous workload simulation
	slog.Info("Starting continuous workload simulation")
	workloadDone := make(chan bool)

	go func() {
		defer func() { workloadDone <- true }()

		for i := range 100 {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Random delay between operations
			time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)

			// Execute ExampleQuery with random data
			_, err := querier.ExampleQuery(ctx, pool, pgtype.Text{
				String: randomString(10),
				Valid:  true,
			})
			if err != nil {
				slog.Error("Failed to execute ExampleQuery in workload", "error", err)
			}

			// Another random delay
			time.Sleep(time.Duration(rand.Intn(2)+1) * time.Second)

			// Execute ExampleQuery2 with random data
			_, err = querier.ExampleQuery2(ctx, pool, pgtype.Text{
				String: randomString(30),
				Valid:  true,
			})
			if err != nil {
				slog.Error("Failed to execute ExampleQuery2 in workload", "error", err)
			}

			if i%10 == 0 {
				slog.Info("Workload progress", "completed_iterations", i)
			}
		}
		slog.Info("Workload simulation completed")
	}()

	// Wait for shutdown signal or workload completion
	select {
	case sig := <-sigChan:
		slog.Info("Received shutdown signal", "signal", sig.String())
	case <-workloadDone:
		slog.Info("Workload completed, keeping server running for monitoring...")
		// Keep running for monitoring purposes
		<-sigChan
	}

	slog.Info("Application shutting down gracefully")
}
