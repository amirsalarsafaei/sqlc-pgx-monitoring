package poolstatus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type PoolStatusSuite struct {
	suite.Suite

	ctx      context.Context
	reader   *metric.ManualReader
	provider *metric.MeterProvider
	stats    *fakeStat
}

func TestPoolStatusSuite(t *testing.T) {
	suite.Run(t, new(PoolStatusSuite))
}

func (s *PoolStatusSuite) SetupTest() {
	s.ctx = context.Background()
	s.reader = metric.NewManualReader()
	s.provider = metric.NewMeterProvider(metric.WithReader(s.reader))
	s.stats = &fakeStat{
		acquiredConns:           5,
		idleConns:               2,
		maxConns:                10,
		constructingConns:       1,
		acquireCount:            100,
		canceledAcquireCount:    5,
		emptyAcquireCount:       20,
		newConnsCount:           8,
		maxLifetimeDestroyCount: 3,
		maxIdleDestroyCount:     4,
		acquireDuration:         1500 * time.Millisecond,
		emptyAcquireWaitTime:    500 * time.Millisecond,
	}
}

func (s *PoolStatusSuite) TestObservesPoolStats() {
	s.Require().NoError(register(s.stats, WithMeterProvider(s.provider)))

	rm := s.collect()

	s.Equal(map[string]float64{
		"db.client.connections.usage{state=used}":         5,
		"db.client.connections.usage{state=idle}":         2,
		"db.client.connection.max":                        10,
		"db.client.connections.pending_requests":          1,
		"pgx.pool.acquires":                               100,
		"pgx.pool.canceled_acquires":                      5,
		"pgx.pool.waited_for_acquires":                    20,
		"pgx.pool.connections.created":                    8,
		"pgx.pool.connections.destroyed{reason=lifetime}": 3,
		"pgx.pool.connections.destroyed{reason=idletime}": 4,
		"pgx.pool.acquire.duration":                       1.5,
		"pgx.pool.acquire.wait.duration":                  0.5,
	}, observations(rm))

	s.Equal(map[string]string{
		"db.client.connections.usage":            "{connection}",
		"db.client.connection.max":               "{connection}",
		"db.client.connections.pending_requests": "{request}",
		"pgx.pool.acquires":                      "{request}",
		"pgx.pool.canceled_acquires":             "{request}",
		"pgx.pool.waited_for_acquires":           "{request}",
		"pgx.pool.connections.created":           "{connection}",
		"pgx.pool.connections.destroyed":         "{connection}",
		"pgx.pool.acquire.duration":              "s",
		"pgx.pool.acquire.wait.duration":         "s",
	}, units(rm))

	s.Equal(map[string]string{
		"db.client.connections.usage":            semconv.DBClientConnectionsUsageDescription,
		"db.client.connection.max":               semconv.DBClientConnectionMaxDescription,
		"db.client.connections.pending_requests": semconv.DBClientConnectionsPendingRequestsDescription,
		"pgx.pool.acquires":                      "Cumulative count of successful acquires from the pool.",
		"pgx.pool.canceled_acquires":             "Cumulative count of acquires from the pool that were canceled by a context.",
		"pgx.pool.waited_for_acquires":           "Cumulative count of acquires that waited for a resource to be released or constructed because the pool was empty.",
		"pgx.pool.connections.created":           "Cumulative count of new connections opened.",
		"pgx.pool.connections.destroyed":         "Cumulative count of connections destroyed, with a reason attribute.",
		"pgx.pool.acquire.duration":              "Total duration of all successful acquires from the pool.",
		"pgx.pool.acquire.wait.duration":         "The cumulative time successful acquires from the pool waited for a resource to be released or constructed because the pool was empty.",
	}, descriptions(rm))

	kinds := map[string]string{}
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch data := m.Data.(type) {
		case metricdata.Gauge[int64]:
			kinds[m.Name] = "int64 gauge"
		case metricdata.Sum[int64]:
			s.True(data.IsMonotonic, m.Name)
			kinds[m.Name] = "int64 counter"
		case metricdata.Sum[float64]:
			s.True(data.IsMonotonic, m.Name)
			kinds[m.Name] = "float64 counter"
		}
	}
	s.Equal(map[string]string{
		"db.client.connections.usage":            "int64 gauge",
		"db.client.connection.max":               "int64 gauge",
		"db.client.connections.pending_requests": "int64 gauge",
		"pgx.pool.acquires":                      "int64 counter",
		"pgx.pool.canceled_acquires":             "int64 counter",
		"pgx.pool.waited_for_acquires":           "int64 counter",
		"pgx.pool.connections.created":           "int64 counter",
		"pgx.pool.connections.destroyed":         "int64 counter",
		"pgx.pool.acquire.duration":              "float64 counter",
		"pgx.pool.acquire.wait.duration":         "float64 counter",
	}, kinds)
}

func (s *PoolStatusSuite) TestReadsStatsOnEveryCollection() {
	s.Require().NoError(register(s.stats, WithMeterProvider(s.provider)))
	s.collect()

	s.stats.acquiredConns = 9
	s.stats.acquireCount = 250

	obs := observations(s.collect())
	s.Equal(float64(9), obs["db.client.connections.usage{state=used}"])
	s.Equal(float64(250), obs["pgx.pool.acquires"])
}

func (s *PoolStatusSuite) TestWithAttributesAccumulate() {
	s.Require().NoError(register(s.stats,
		WithMeterProvider(s.provider),
		WithAttributes(attribute.String("pool", "primary")),
		WithAttributes(attribute.String("service.name", "users")),
	))

	obs := observations(s.collect())

	s.Len(obs, 12)
	for name := range obs {
		s.Contains(name, "pool=primary", name)
		s.Contains(name, "service.name=users", name)
	}
	s.Contains(obs, "db.client.connections.usage{pool=primary,service.name=users,state=used}")
	s.Contains(obs, "pgx.pool.acquires{pool=primary,service.name=users}")
}

func (s *PoolStatusSuite) TestRegisterFailsWhenInstrumentCreationFails() {
	err := register(s.stats, WithMeterProvider(&erroringMeterProvider{err: errors.New("meter closed")}))

	s.ErrorContains(err, "failed to create usage metric: meter closed")
}

func (s *PoolStatusSuite) collect() metricdata.ResourceMetrics {
	s.T().Helper()

	var rm metricdata.ResourceMetrics
	s.Require().NoError(s.reader.Collect(s.ctx, &rm))
	s.Require().Len(rm.ScopeMetrics, 1)
	s.Equal(instrumentationName, rm.ScopeMetrics[0].Scope.Name)

	return rm
}

func observations(rm metricdata.ResourceMetrics) map[string]float64 {
	obs := map[string]float64{}
	add := func(name string, attrs attribute.Set, value float64) {
		if enc := attrs.Encoded(attribute.DefaultEncoder()); enc != "" {
			name += "{" + enc + "}"
		}
		obs[name] = value
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Gauge[int64]:
				for _, p := range data.DataPoints {
					add(m.Name, p.Attributes, float64(p.Value))
				}
			case metricdata.Sum[int64]:
				for _, p := range data.DataPoints {
					add(m.Name, p.Attributes, float64(p.Value))
				}
			case metricdata.Sum[float64]:
				for _, p := range data.DataPoints {
					add(m.Name, p.Attributes, p.Value)
				}
			}
		}
	}

	return obs
}

func units(rm metricdata.ResourceMetrics) map[string]string {
	u := map[string]string{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			u[m.Name] = m.Unit
		}
	}

	return u
}

func descriptions(rm metricdata.ResourceMetrics) map[string]string {
	d := map[string]string{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			d[m.Name] = m.Description
		}
	}

	return d
}

type fakeStat struct {
	acquireCount            int64
	acquireDuration         time.Duration
	acquiredConns           int32
	canceledAcquireCount    int64
	constructingConns       int32
	emptyAcquireCount       int64
	emptyAcquireWaitTime    time.Duration
	idleConns               int32
	maxConns                int32
	maxIdleDestroyCount     int64
	maxLifetimeDestroyCount int64
	newConnsCount           int64
	totalConns              int32
}

func (f *fakeStat) Stat() Stat { return f }

func (f *fakeStat) AcquireCount() int64                 { return f.acquireCount }
func (f *fakeStat) AcquireDuration() time.Duration      { return f.acquireDuration }
func (f *fakeStat) AcquiredConns() int32                { return f.acquiredConns }
func (f *fakeStat) CanceledAcquireCount() int64         { return f.canceledAcquireCount }
func (f *fakeStat) ConstructingConns() int32            { return f.constructingConns }
func (f *fakeStat) EmptyAcquireCount() int64            { return f.emptyAcquireCount }
func (f *fakeStat) EmptyAcquireWaitTime() time.Duration { return f.emptyAcquireWaitTime }
func (f *fakeStat) IdleConns() int32                    { return f.idleConns }
func (f *fakeStat) MaxConns() int32                     { return f.maxConns }
func (f *fakeStat) MaxIdleDestroyCount() int64          { return f.maxIdleDestroyCount }
func (f *fakeStat) MaxLifetimeDestroyCount() int64      { return f.maxLifetimeDestroyCount }
func (f *fakeStat) NewConnsCount() int64                { return f.newConnsCount }
func (f *fakeStat) TotalConns() int32                   { return f.totalConns }

type erroringMeterProvider struct {
	otelmetric.MeterProvider
	err error
}

func (p *erroringMeterProvider) Meter(string, ...otelmetric.MeterOption) otelmetric.Meter {
	return &erroringMeter{err: p.err}
}

type erroringMeter struct {
	otelmetric.Meter
	err error
}

func (m *erroringMeter) Int64ObservableGauge(string, ...otelmetric.Int64ObservableGaugeOption) (otelmetric.Int64ObservableGauge, error) {
	return nil, m.err
}
