package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/internal/example/db"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/internal/example/db/entities/exampletable"
)

// func setupMetrics() (*prometheus.Exporter, error) {
// 	resource := resource.NewWithAttributes(
// 		semconv.SchemaURL,
// 		semconv.ServiceName("example-service"),
// 		semconv.ServiceVersion("1.0.0"),
// 	)
//
// 	exporter, err := prometheus.New()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
// 	}
//
// 	provider := sdkmetric.NewMeterProvider(
// 		sdkmetric.WithResource(resource),
// 		sdkmetric.WithReader(exporter),
// 		sdkmetric.WithView(sdkmetric.NewView(
// 			sdkmetric.Instrument{
// 				Name: "*",
// 				Kind: sdkmetric.InstrumentKindHistogram,
// 			},
// 			sdkmetric.Stream{
// 				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
// 					Boundaries: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
// 				},
// 			},
// 		)),
// 	)
//
// 	otel.SetMeterProvider(provider)
// 	return exporter, nil
// }

// func startMetricsServer(exporter *prometheus.Exporter) *http.Server {
// 	mux := http.NewServeMux()
// 	mux.Handle("/metrics", promhttp.Handler())
//
// 	server := &http.Server{
// 		Addr:              ":9000",
// 		Handler:           mux,
// 		ReadHeaderTimeout: 100 * time.Millisecond,
// 	}
//
// 	go func() {
// 		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 			panic(fmt.Sprintf("metrics server error: %v", err))
// 		}
// 	}()
//
// 	return server
// }

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

	pool, err := db.GetConnectionPool(
		ctx,
		db.DBConfig{
			User: "example",
			Pwd:  "complex-password",
			Host: "localhost",
			Port: "5432",
			Name: "example_db",
		},
	)
	if err != nil {
		panic(err)
	}

	querier := exampletable.New()
	exampleRows, err := querier.ExampleQuery(ctx, pool, pgtype.Text{
		String: "basdf",
		Valid:  true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(len(exampleRows))

	for i, row := range exampleRows {
		fmt.Printf("%d row %s\n", i, row.Foo.String)
	}

	_, err = querier.ExampleQuery2(ctx, pool, pgtype.Text{
		String: "basdf",
		Valid:  true,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 50; i++ {
		time.Sleep(2 * time.Second)
		_, err := querier.ExampleQuery(ctx, pool, pgtype.Text{
			String: randomString(10),
			Valid:  true,
		})
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Duration(rand.Intn(3)) * time.Second)
		_, err = querier.ExampleQuery2(ctx, pool, pgtype.Text{
			String: randomString(30),
			Valid:  true,
		})
		if err != nil {
			panic(err)
		}

	}

	time.Sleep(time.Hour)
}
