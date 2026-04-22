package opentelemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentationName is the instrumentation scope name used by the Tracer
// and Meter exposed by this package.
const InstrumentationName = "github.com/get-eventually/go-eventually/opentelemetry"

// MetricUnitMilliseconds is the [OpenTelemetry-compatible unit] used by the
// duration histograms exposed by this package.
//
// [OpenTelemetry-compatible unit]: https://ucum.org/ucum
const MetricUnitMilliseconds = "ms"

type config struct {
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider
}

func (c config) meter() metric.Meter {
	return c.MeterProvider.Meter(InstrumentationName)
}

func (c config) tracer() trace.Tracer {
	return c.TracerProvider.Tracer(InstrumentationName)
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type meterProviderOption struct{ metric.MeterProvider }

func (o meterProviderOption) apply(c *config) {
	c.MeterProvider = o.MeterProvider
}

// WithMeterProvider specifies the metric.MeterProvider instance to use for the instrumentation.
// By default, the global metric.MeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return meterProviderOption{provider}
}

type tracerProviderOption struct{ trace.TracerProvider }

func (o tracerProviderOption) apply(c *config) {
	c.TracerProvider = o.TracerProvider
}

// WithTracerProvider specifies the trace.TracerProvider instance to use for the instrumentation.
// By default, the global trace.TracerProvider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return tracerProviderOption{provider}
}

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider:  otel.GetMeterProvider(),
		TracerProvider: otel.GetTracerProvider(),
	}

	for _, opt := range opts {
		opt.apply(&c)
	}

	return c
}
