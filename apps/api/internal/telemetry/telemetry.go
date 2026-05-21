package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName string
	OTLPEndpoint string
	// Disabled forces tracing off even when OTLP is configured.
	Disabled bool
}

var (
	initialized bool
	tracerProvider *sdktrace.TracerProvider
)

func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.Disabled || strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return func(context.Context) error { return nil }, nil
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "codeatlas-api"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}
	if endpoint != "" {
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpointURL(endpoint),
		}
		if strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "true") {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("otlp exporter: %w", err)
		}
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	tracerProvider = tp
	initialized = true

	return tp.Shutdown, nil
}

func Enabled() bool {
	return initialized
}

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

func PGXTracer() *otelpgx.Tracer {
	return otelpgx.NewTracer()
}

func HTTPMiddleware(operation string, next http.Handler) http.Handler {
	if !initialized {
		return next
	}
	return otelhttp.NewHandler(next, operation,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

func Start(ctx context.Context, tracerName, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !initialized {
		return ctx, trace.SpanFromContext(ctx)
	}
	ctx, span := Tracer(tracerName).Start(ctx, spanName)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

func RecordError(span trace.Span, err error) {
	if !initialized || err == nil || span == nil {
		return
	}
	span.RecordError(err)
}

func ShutdownTimeout(parent context.Context, shutdown func(context.Context) error, timeout time.Duration) error {
	if shutdown == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return shutdown(ctx)
}
