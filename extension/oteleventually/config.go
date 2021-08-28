package oteleventually

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/get-eventually/go-eventually/extension/oteleventually"

type config struct {
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider
	ProjectionName string
}

func (c config) meter() metric.Meter {
	return c.MeterProvider.Meter(instrumentationName)
}

func (c config) tracer() trace.Tracer {
	return c.TracerProvider.Tracer(instrumentationName)
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

type projectionNameOption string

func (o projectionNameOption) apply(c *config) {
	c.ProjectionName = string(o)
}

// WithProjectionName specifies the name displayed in spans and metrics reported by the instrumentation.
func WithProjectionName(name string) Option {
	return projectionNameOption(name)
}

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider:  global.GetMeterProvider(),
		TracerProvider: otel.GetTracerProvider(),
	}

	for _, opt := range opts {
		opt.apply(&c)
	}

	return c
}
