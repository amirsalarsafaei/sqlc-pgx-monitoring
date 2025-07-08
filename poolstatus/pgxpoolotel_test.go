package poolstatus_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type mockStat struct {
	acquireCountVal            int64
	acquireDurationVal         time.Duration
	acquiredConnsVal           int32
	canceledAcquireCountVal    int64
	constructingConnsVal       int32
	emptyAcquireCountVal       int64
	emptyAcquireWaitTimeVal    time.Duration
	idleConnsVal               int32
	maxConnsVal                int32
	maxIdleDestroyCountVal     int64
	maxLifetimeDestroyCountVal int64
	newConnsCountVal           int64
	totalConnsVal              int32
}

// Implement the poolstatus.Stat interface
func (m *mockStat) AcquireCount() int64                   { return m.acquireCountVal }
func (m *mockStat) AcquireDuration() time.Duration        { return m.acquireDurationVal }
func (m *mockStat) AcquiredConns() int32                  { return m.acquiredConnsVal }
func (m *mockStat) CanceledAcquireCount() int64             { return m.canceledAcquireCountVal }
func (m *mockStat) ConstructingConns() int32              { return m.constructingConnsVal }
func (m *mockStat) EmptyAcquireCount() int64                { return m.emptyAcquireCountVal }
func (m *mockStat) EmptyAcquireWaitTime() time.Duration     { return m.emptyAcquireWaitTimeVal }
func (m *mockStat) IdleConns() int32                      { return m.idleConnsVal }
func (m *mockStat) MaxConns() int32                       { return m.maxConnsVal }
func (m *mockStat) MaxIdleDestroyCount() int64            { return m.maxIdleDestroyCountVal }
func (m *mockStat) MaxLifetimeDestroyCount() int64        { return m.maxLifetimeDestroyCountVal }
func (m *mockStat) NewConnsCount() int64                  { return m.newConnsCountVal }
func (m *mockStat) TotalConns() int32                     { return m.totalConnsVal }

// mockStater implements the poolstatus.Stater interface for testing.
type mockStater struct {
	stats poolstatus.Stat
}

// Stat returns the mock stats, satisfying the poolstatus.Stater interface.
func (m *mockStater) Stat() poolstatus.Stat {
	return m.stats
}

// PoolStatusTestSuite groups all tests for the poolstatus package.
type PoolStatusTestSuite struct {
	suite.Suite
	reader   *metric.ManualReader
	provider *metric.MeterProvider
	stater   poolstatus.Stater
	stats    *mockStat
	ctx      context.Context
}

func (s *PoolStatusTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.reader = metric.NewManualReader()
	s.provider = metric.NewMeterProvider(metric.WithReader(s.reader))

	s.stats = &mockStat{
		acquiredConnsVal:           5,
		idleConnsVal:               2,
		maxConnsVal:                10,
		constructingConnsVal:       1,
		acquireCountVal:            100,
		canceledAcquireCountVal:    5,
		emptyAcquireCountVal:       20,
		newConnsCountVal:           8,
		maxLifetimeDestroyCountVal: 3,
		maxIdleDestroyCountVal:     4,
		acquireDurationVal:         1 * time.Second,
		emptyAcquireWaitTimeVal:    500 * time.Millisecond,
	}

	s.stater = &mockStater{stats: s.stats}
}

func TestPoolStatusSuite(t *testing.T) {
	suite.Run(t, new(PoolStatusTestSuite))
}


func (s *PoolStatusTestSuite) TestRegisterMetrics_DefaultOptions() {
	err := poolstatus.Register(s.stater, poolstatus.WithMeterProvider(s.provider))
	s.Require().NoError(err, "Register should not return an error")

	rm := s.collectMetrics()
	s.assertAllMetrics(rm.ScopeMetrics[0], s.stats, nil)
}

func (s *PoolStatusTestSuite) TestRegisterMetrics_WithAttributes() {
	commonAttrs := []attribute.KeyValue{
		attribute.String("db.instance", "test-db"),
		attribute.String("service.name", "my-app"),
	}

	err := poolstatus.Register(s.stater,
		poolstatus.WithMeterProvider(s.provider),
		poolstatus.WithAttributes(commonAttrs...),
	)
	s.Require().NoError(err, "Register should not return an error")

	rm := s.collectMetrics()
	s.assertAllMetrics(rm.ScopeMetrics[0], s.stats, commonAttrs)
}

func (s *PoolStatusTestSuite) TestRegisterMetrics_ErrorOnRegistration() {
	mockProvider := &erroringMeterProvider{err: errors.New("registration failed")}

	err := poolstatus.Register(s.stater, poolstatus.WithMeterProvider(mockProvider))

	s.Require().Error(err)
	s.Assert().ErrorContains(err, "failed to create usage metric")
	s.Assert().ErrorContains(err, "registration failed")
}


func (s *PoolStatusTestSuite) collectMetrics() metricdata.ResourceMetrics {
	s.T().Helper()
	var rm metricdata.ResourceMetrics
	err := s.reader.Collect(s.ctx, &rm)
	s.Require().NoError(err, "Collect should not return an error")
	s.Require().Len(rm.ScopeMetrics, 1, "should have one scope metric")
	return rm
}

func (s *PoolStatusTestSuite) assertAllMetrics(scopeMetrics metricdata.ScopeMetrics, stats poolstatus.Stat, commonAttrs []attribute.KeyValue) {
	s.T().Helper()
	assert.Equal(s.T(), "github.com/amirsalarsafaei/sqlc-pgx-monitoring", scopeMetrics.Scope.Name)
	require.NotEmpty(s.T(), scopeMetrics.Metrics, "should have registered metrics")

	metrics := scopeMetrics.Metrics

	s.assertUsageMetric(metrics, stats, commonAttrs)
	s.assertMaxConnsMetric(metrics, stats, commonAttrs)
	s.assertPendingMetric(metrics, stats, commonAttrs)
	s.assertAcquireCountMetric(metrics, stats, commonAttrs)
	s.assertCanceledAcquireCountMetric(metrics, stats, commonAttrs)
	s.assertWaitedForAcquireCountMetric(metrics, stats, commonAttrs)
	s.assertConnsCreatedMetric(metrics, stats, commonAttrs)
	s.assertConnsDestroyedMetric(metrics, stats, commonAttrs)
	s.assertAcquireDurationMetric(metrics, stats, commonAttrs)
	s.assertWaitedForAcquireDurationMetric(metrics, stats, commonAttrs)
}

func (s *PoolStatusTestSuite) findMetric(name string, metrics []metricdata.Metrics) metricdata.Metrics {
	s.T().Helper()
	for _, m := range metrics {
		if m.Name == name {
			return m
		}
	}
	s.Require().Fail("metric not found", "metric with name '%s' was not found", name)
	return metricdata.Metrics{} // Unreachable
}

func (s *PoolStatusTestSuite) assertCommonAttributes(pointAttrs attribute.Set, commonAttrs []attribute.KeyValue) {
	s.T().Helper()
	if commonAttrs == nil {
		return // Nothing to check
	}
	for _, attr := range commonAttrs {
		val, ok := pointAttrs.Value(attr.Key)
		s.Require().True(ok, "common attribute '%s' not found", attr.Key)
		s.Assert().Equal(attr.Value, val, "common attribute value for '%s' does not match", attr.Key)
	}
}

func (s *PoolStatusTestSuite) assertUsageMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric(semconv.DBClientConnectionsUsageName, metrics)
	s.Assert().Equal(semconv.DBClientConnectionsUsageDescription, m.Description)
	s.Assert().Equal(semconv.DBClientConnectionsUsageUnit, m.Unit)

	gauge, ok := m.Data.(metricdata.Gauge[int64])
	s.Require().True(ok, "metric '%s' should be a Gauge[int64]", m.Name)
	s.Require().Len(gauge.DataPoints, 2, "usage metric should have 2 data points (used, idle)")

	var idlePoint, usedPoint metricdata.DataPoint[int64]
	for _, p := range gauge.DataPoints {
		if state, ok := p.Attributes.Value("state"); ok {
			if state.AsString() == "idle" {
				idlePoint = p
			} else if state.AsString() == "used" {
				usedPoint = p
			}
		}
	}
	s.Require().NotNil(idlePoint.Attributes, "idle data point not found")
	s.Assert().Equal(int64(stats.IdleConns()), idlePoint.Value)
	s.assertCommonAttributes(idlePoint.Attributes, attrs)

	s.Require().NotNil(usedPoint.Attributes, "used data point not found")
	s.Assert().Equal(int64(stats.AcquiredConns()), usedPoint.Value)
	s.assertCommonAttributes(usedPoint.Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertMaxConnsMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric(semconv.DBClientConnectionMaxName, metrics)
	s.Assert().Equal(semconv.DBClientConnectionMaxDescription, m.Description)
	gauge := s.getGaugeDataPoints(m)
	s.Assert().Equal(int64(stats.MaxConns()), gauge[0].Value)
	s.assertCommonAttributes(gauge[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertPendingMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric(semconv.DBClientConnectionsPendingRequestsName, metrics)
	s.Assert().Equal(semconv.DBClientConnectionsPendingRequestsDescription, m.Description)
	gauge := s.getGaugeDataPoints(m)
	s.Assert().Equal(int64(stats.ConstructingConns()), gauge[0].Value)
	s.assertCommonAttributes(gauge[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertAcquireCountMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.acquires", metrics)
	s.Assert().Equal("Cumulative count of successful acquires from the pool.", m.Description)
	sum := s.getIntSumDataPoints(m)
	s.Assert().Equal(stats.AcquireCount(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertCanceledAcquireCountMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.canceled_acquires", metrics)
	s.Assert().Equal("Cumulative count of acquires from the pool that were canceled by a context.", m.Description)
	sum := s.getIntSumDataPoints(m)
	s.Assert().Equal(stats.CanceledAcquireCount(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertWaitedForAcquireCountMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.waited_for_acquires", metrics)
	s.Assert().Equal("Cumulative count of acquires that waited for a resource to be released or constructed because the pool was empty.", m.Description)
	sum := s.getIntSumDataPoints(m)
	s.Assert().Equal(stats.EmptyAcquireCount(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertConnsCreatedMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.connections.created", metrics)
	s.Assert().Equal("Cumulative count of new connections opened.", m.Description)
	sum := s.getIntSumDataPoints(m)
	s.Assert().Equal(stats.NewConnsCount(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertAcquireDurationMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.acquire.duration", metrics)
	s.Assert().Equal("Total duration of all successful acquires from the pool.", m.Description)
	sum := s.getFloatSumDataPoints(m)
	s.Assert().Equal(stats.AcquireDuration().Seconds(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertWaitedForAcquireDurationMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.acquire.wait.duration", metrics)
	s.Assert().Equal("The cumulative time successful acquires from the pool waited for a resource to be released or constructed because the pool was empty.", m.Description)
	sum := s.getFloatSumDataPoints(m)
	s.Assert().Equal(stats.EmptyAcquireWaitTime().Seconds(), sum[0].Value)
	s.assertCommonAttributes(sum[0].Attributes, attrs)
}

func (s *PoolStatusTestSuite) assertConnsDestroyedMetric(metrics []metricdata.Metrics, stats poolstatus.Stat, attrs []attribute.KeyValue) {
	s.T().Helper()
	m := s.findMetric("pgx.pool.connections.destroyed", metrics)
	s.Assert().Equal("Cumulative count of connections destroyed, with a reason attribute.", m.Description)
	s.Assert().Equal("{connection}", m.Unit)

	sum, ok := m.Data.(metricdata.Sum[int64])
	s.Require().True(ok, "metric '%s' should be a Sum[int64]", m.Name)
	s.Require().Len(sum.DataPoints, 2, "destroyed metric should have 2 data points (lifetime, idletime)")

	var lifetimePoint, idlePoint metricdata.DataPoint[int64]
	for _, p := range sum.DataPoints {
		if reason, ok := p.Attributes.Value("reason"); ok {
			if reason.AsString() == "lifetime" {
				lifetimePoint = p
			} else if reason.AsString() == "idletime" {
				idlePoint = p
			}
		}
	}
	s.Require().NotNil(lifetimePoint.Attributes, "lifetime point not found")
	s.Assert().Equal(stats.MaxLifetimeDestroyCount(), lifetimePoint.Value)
	s.assertCommonAttributes(lifetimePoint.Attributes, attrs)

	s.Require().NotNil(idlePoint.Attributes, "idletime point not found")
	s.Assert().Equal(stats.MaxIdleDestroyCount(), idlePoint.Value)
	s.assertCommonAttributes(idlePoint.Attributes, attrs)
}

func (s *PoolStatusTestSuite) getGaugeDataPoints(m metricdata.Metrics) []metricdata.DataPoint[int64] {
	s.T().Helper()
	gauge, ok := m.Data.(metricdata.Gauge[int64])
	s.Require().True(ok, "metric '%s' should be a Gauge[int64]", m.Name)
	s.Require().Len(gauge.DataPoints, 1)
	return gauge.DataPoints
}

func (s *PoolStatusTestSuite) getIntSumDataPoints(m metricdata.Metrics) []metricdata.DataPoint[int64] {
	s.T().Helper()
	sum, ok := m.Data.(metricdata.Sum[int64])
	s.Require().True(ok, "metric '%s' should be a Sum[int64]", m.Name)
	s.Require().Len(sum.DataPoints, 1)
	return sum.DataPoints
}

func (s *PoolStatusTestSuite) getFloatSumDataPoints(m metricdata.Metrics) []metricdata.DataPoint[float64] {
	s.T().Helper()
	sum, ok := m.Data.(metricdata.Sum[float64])
	s.Require().True(ok, "metric '%s' should be a Sum[float64]", m.Name)
	s.Require().Len(sum.DataPoints, 1)
	return sum.DataPoints
}

type erroringMeterProvider struct {
	otelmetric.MeterProvider
	err error
}

func (p *erroringMeterProvider) Meter(name string, opts ...otelmetric.MeterOption) otelmetric.Meter {
	return &erroringMeter{err: p.err}
}

type erroringMeter struct {
	otelmetric.Meter
	err error
}

func (m *erroringMeter) Int64ObservableGauge(name string, options ...otelmetric.Int64ObservableGaugeOption) (otelmetric.Int64ObservableGauge, error) {
	return nil, m.err
}
