package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/internal/example/db"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/internal/example/db/entities/exampletable"
)

func getPrometheusServer(port int) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 100 * time.Millisecond,
	}
}

func servePrometheus(prometheusServer *http.Server) {
	if err := prometheusServer.ListenAndServe(); err != nil {
		panic(err)
	}
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
	err := db.Migrate("postgres://example:complex-password@localhost:5432/example_db?sslmode=disable")
	if err != nil {
		panic(err)
	}

	go servePrometheus(getPrometheusServer(9000))

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
