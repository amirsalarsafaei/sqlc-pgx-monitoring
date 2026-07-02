//go:build integration

package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
)

type TelemetrySuite struct {
	dbSuite
}

func TestTelemetrySuite(t *testing.T) {
	suite.Run(t, new(TelemetrySuite))
}

func (s *TelemetrySuite) TestQuerySpanCarriesOperationName() {
	s.mustCreateUser("telemetry", "telemetry@example.com")

	spans := s.spansByName("postgresql.query CreateUser")
	s.Require().Len(spans, 1)

	attrs := spanAttributes(spans[0])
	s.Equal("CreateUser", attrs[semconv.DBOperationNameKey].AsString())
	s.Equal("CreateUser", attrs[dbtracer.SQLCQueryNameKey].AsString())
	s.Equal(codes.Ok, spans[0].Status().Code)
}

func (s *TelemetrySuite) TestQueryErrorMarksSpan() {
	s.mustCreateUser("dup", "dup@example.com")

	_, err := s.querier.CreateUser(s.ctx, s.pool, "dup", "another@example.com")
	s.Require().Error(err)

	spans := s.spansByName("postgresql.query CreateUser")
	s.Require().Len(spans, 2)
	failed := spans[1]
	s.Equal(codes.Error, failed.Status().Code)
	s.NotEmpty(spanAttributes(failed)[dbtracer.DBStatusCodeKey].AsString())
}

func (s *TelemetrySuite) TestLatencyHistogramRecorded() {
	s.mustCreateUser("metrics", "metrics@example.com")

	var rm metricdata.ResourceMetrics
	s.Require().NoError(s.metricReader.Collect(s.ctx, &rm))

	point := s.histogramPointFor(rm, "CreateUser")
	s.Require().NotNil(point, "expected a duration data point for CreateUser")
	s.Equal(uint64(1), point.Count)
}

func (s *TelemetrySuite) histogramPointFor(rm metricdata.ResourceMetrics, queryName string) *metricdata.HistogramDataPoint[float64] {
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			hist, ok := metric.Data.(metricdata.Histogram[float64])
			if !ok {
				continue
			}
			for i := range hist.DataPoints {
				point := &hist.DataPoints[i]
				if value, ok := point.Attributes.Value(dbtracer.SQLCQueryNameKey); ok && value.AsString() == queryName {
					return point
				}
			}
		}
	}
	return nil
}
