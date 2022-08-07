package app

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/pyroscope-io/client/pyroscope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

type Observability struct {
	TracerProvider *trace.TracerProvider
	Registry       *prometheus.Registry
	Options        ObservabilityOptions
}

type ObservabilityOptions struct {
	Debug             bool
	TracerServiceName string
	ProfilerName      string
	ProfilerAddr      string
}

func newObservability(ctx context.Context, o ObservabilityOptions) (Observability, error) {
	obs := Observability{
		Options:  o,
		Registry: prometheus.NewRegistry(),
	}

	obs.registerCollectors()

	tp, err := obs.getTracerProvider(ctx)
	if err != nil {
		return obs, fmt.Errorf("unable to get tracer provider: %w", err)
	}
	obs.TracerProvider = tp

	if o.Debug {
		obs.startProfiler()
	}

	return obs, nil
}

func (o *Observability) getTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	cl := otlptracehttp.NewClient(otlptracehttp.WithInsecure())
	exp, err := otlptrace.New(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("unable to create new exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("",
			semconv.ServiceNameKey.String(o.Options.TracerServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create new resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetErrorHandler(noopErrorHandler())

	return tp, nil
}

func (o *Observability) startProfiler() {
	pyroscope.Start(pyroscope.Config{
		ApplicationName: o.Options.ProfilerName,
		ServerAddress:   o.Options.ProfilerAddr,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
		},
	})
}

func (o *Observability) registerCollectors() {
	o.Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	o.Registry.MustRegister(collectors.NewGoCollector())
}

func noopErrorHandler() otel.ErrorHandlerFunc {
	return func(error) {}
}
