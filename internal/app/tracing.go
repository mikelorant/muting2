package app

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

type TracingOptions struct {
	ServiceName string
}

func NewTracingProvider(ctx context.Context, o TracingOptions) (*trace.TracerProvider, error) {
	cl := otlptracehttp.NewClient(otlptracehttp.WithInsecure())
	exp, err := otlptrace.New(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("unable to create new exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("",
			semconv.ServiceNameKey.String(o.ServiceName),
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

	return tp, nil
}
