package prometheustools

import (
	"github.com/prometheus/client_golang/prometheus"
)

type histogram struct {
	vec         *prometheus.HistogramVec
	labels      []string
	labelValues map[string]string
}

func NewHistogram(name, help string, buckets []float64, registerer prometheus.Registerer, labels ...string) Observer {
	result := &histogram{
		vec: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		}, labels),

		labels:      labels,
		labelValues: make(map[string]string),
	}

	registerer.MustRegister(result.vec)

	return result
}

func (s *histogram) Observe(value float64) {
	values := make([]string, len(s.labels))

	for i, label := range s.labels {
		values[i] = s.labelValues[label]
	}

	s.vec.WithLabelValues(values...).Observe(value)
}

func (s *histogram) With(labels map[string]string) Observer {
	newLabelValues := make(map[string]string)

	for label, value := range s.labelValues {
		newLabelValues[label] = value
	}

	for label, value := range labels {
		newLabelValues[label] = value
	}

	return &histogram{
		labelValues: newLabelValues,
		labels:      s.labels,
		vec:         s.vec,
	}
}
