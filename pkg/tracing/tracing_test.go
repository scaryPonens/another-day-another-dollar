package tracing

import (
	"context"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInitTracerDisabled(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "false")
	tp, tracer, err := InitTracer(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil || tracer == nil {
		t.Fatal("expected tracer provider")
	}
}

func TestInitTracerEnabledWithStubExporter(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")

	orig := newTraceExporter
	defer func() { newTraceExporter = orig }()

	stub := &stubExporter{}
	newTraceExporter = func(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
		stub.endpoint = endpoint
		return stub, nil
	}

	tp, tracer, err := InitTracer(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracer == nil {
		t.Fatal("expected tracer")
	}
	if stub.endpoint != "collector:4317" {
		t.Fatalf("expected endpoint to be propagated, got %s", stub.endpoint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

type stubExporter struct {
	endpoint string
}

func (s *stubExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (s *stubExporter) Shutdown(ctx context.Context) error {
	return nil
}
