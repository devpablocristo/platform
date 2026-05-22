package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig configura el TracerProvider compartido.
type TracingConfig struct {
	// ServiceName se mapea a service.name (resource attribute).
	ServiceName string
	// ServiceVersion se mapea a service.version. Si vacío, "0.0.0".
	ServiceVersion string
	// Environment se mapea a deployment.environment.name (dev, staging, prod).
	Environment string

	// Exporter determina el destino. Valores:
	//   "otlp"   → OTLP HTTP (recomendado para prod, requiere OTLPEndpoint).
	//   "stdout" → JSON spans a stdout (útil en dev sin collector).
	//   "none"   → no-op TracerProvider (tracing deshabilitado).
	// Si vacío, default "none".
	Exporter string

	// OTLPEndpoint solo aplica cuando Exporter="otlp" (ej. "localhost:4318").
	OTLPEndpoint string
	// OTLPInsecure usa HTTP en lugar de HTTPS. Útil en dev local.
	OTLPInsecure bool

	// SampleRatio en [0.0, 1.0]. Default 1.0 cuando hay exporter activo.
	SampleRatio float64
}

// NewTracerProvider construye y registra el TracerProvider global según cfg.
// Devuelve una función shutdown que el caller debe invocar al terminar
// (típicamente con un context con timeout en main al recibir SIGTERM).
//
// Si cfg.Exporter es "none" o vacío, registra un TracerProvider no-op y la
// función shutdown es no-op. Esto permite mantener el cableado en código sin
// emitir spans hasta que se configure por env.
func NewTracerProvider(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	cfg = normalizeTracingConfig(cfg)

	noop := func(context.Context) error { return nil }

	if cfg.Exporter == "none" {
		return noop, nil
	}

	exporter, err := buildSpanExporter(ctx, cfg)
	if err != nil {
		return noop, fmt.Errorf("observability: build span exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("deployment.environment.name", cfg.Environment),
		),
	)
	if err != nil {
		return noop, fmt.Errorf("observability: build resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(2*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer devuelve el tracer compartido para el servicio dado. Reusa el
// TracerProvider global registrado por NewTracerProvider.
func Tracer(name string) trace.Tracer {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "platform/observability/go"
	}
	return otel.Tracer(name)
}

// TraceIDFromContext devuelve el trace ID activo en hex, o "" si no hay span.
// Útil para enriquecer logs estructurados.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// SpanIDFromContext devuelve el span ID activo en hex, o "" si no hay span.
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

func buildSpanExporter(ctx context.Context, cfg TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	case "otlp":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	default:
		return nil, errors.New("unknown exporter")
	}
}

func normalizeTracingConfig(cfg TracingConfig) TracingConfig {
	cfg.ServiceName = strings.TrimSpace(cfg.ServiceName)
	if cfg.ServiceName == "" {
		cfg.ServiceName = "unknown"
	}
	cfg.ServiceVersion = strings.TrimSpace(cfg.ServiceVersion)
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "0.0.0"
	}
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	if cfg.Environment == "" {
		cfg.Environment = "local"
	}
	cfg.Exporter = strings.ToLower(strings.TrimSpace(cfg.Exporter))
	if cfg.Exporter == "" {
		cfg.Exporter = "none"
	}
	if cfg.SampleRatio <= 0 || cfg.SampleRatio > 1 {
		cfg.SampleRatio = 1.0
	}
	return cfg
}
