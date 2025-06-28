package dbtracer

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ShouldLog func(err error) bool

type optionCtx struct {
	name                   string
	shouldLog              ShouldLog
	meterProvider          metric.MeterProvider
	traceProvider          trace.TracerProvider
	latencyHistogramConfig struct {
		name        string
		unit        string
		description string
	}
	logger                *slog.Logger
	logArgs               bool
	logArgsLenLimit       int
	includeSQLText        bool
	appendQueryNameToSpan bool
}

type Option func(*optionCtx)

func WithShouldLog(shouldLog ShouldLog) Option {
	return func(oc *optionCtx) {
		oc.shouldLog = shouldLog
	}
}

func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(oc *optionCtx) {
		oc.meterProvider = mp
	}
}

func WithLatencyHistogramConfig(name, unit, description string) Option {
	return func(oc *optionCtx) {
		oc.latencyHistogramConfig.name = name
		oc.latencyHistogramConfig.unit = unit
		oc.latencyHistogramConfig.description = description
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(oc *optionCtx) {
		oc.logger = logger
	}
}

func WithTraceProvider(tp trace.TracerProvider) Option {
	return func(oc *optionCtx) {
		oc.traceProvider = tp
	}
}

func WithLogArgs(enabled bool) Option {
	return func(oc *optionCtx) {
		oc.logArgs = enabled
	}
}

func WithLogArgsLenLimit(limit int) Option {
	return func(oc *optionCtx) {
		oc.logArgsLenLimit = limit
	}
}

func WithIncludeSQLText(includeSQLText bool) Option {
	return func(oc *optionCtx) {
		oc.includeSQLText = includeSQLText
	}
}

func WithAppendQueryNameToSpan(enabled bool) Option {
	return func(oc *optionCtx) {
		oc.appendQueryNameToSpan = enabled
	}
}
