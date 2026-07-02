//go:build integration

package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"

	"examples/db/entities/integration_test"
)

type BatchSuite struct {
	dbSuite
}

func TestBatchSuite(t *testing.T) {
	suite.Run(t, new(BatchSuite))
}

// TestSpansCarryOperationName is the end-to-end check for the batch span naming
// improvement: the batch span and each queued-query span are named after the
// sqlc operation instead of the generic "postgresql.batch[.query]".
func (s *BatchSuite) TestSpansCarryOperationName() {
	params := []integration_test.InsertUsersParams{
		{Username: "batch1", Email: "batch1@example.com"},
		{Username: "batch2", Email: "batch2@example.com"},
	}
	s.execInsertUsers(params)

	batchSpans := s.spansByName("postgresql.batch InsertUsers")
	s.Require().Len(batchSpans, 1)

	querySpans := s.spansByName("postgresql.batch.query InsertUsers")
	s.Len(querySpans, len(params))

	attrs := spanAttributes(batchSpans[0])
	s.Equal("batch", attrs[dbtracer.PGXOperationTypeKey].AsString())
	s.Equal("InsertUsers", attrs[semconv.DBOperationNameKey].AsString())
	s.Equal("InsertUsers", attrs[dbtracer.SQLCQueryNameKey].AsString())
	s.Equal("batchexec", attrs[dbtracer.SQLCQueryCommandKey].AsString())
}

func (s *BatchSuite) execInsertUsers(params []integration_test.InsertUsersParams) {
	s.T().Helper()

	results := s.querier.InsertUsers(s.ctx, s.pool, params)
	var execErr error
	results.Exec(func(_ int, err error) {
		if err != nil {
			execErr = err
		}
	})
	s.Require().NoError(execErr)
	s.Require().NoError(results.Close())
}
