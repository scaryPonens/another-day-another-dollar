package service

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type mockTracer struct{ trace.Tracer }

func (m mockTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, mockSpan{}
}

type mockSpan struct{ trace.Span }

func (m mockSpan) End(options ...trace.SpanEndOption) {}

func TestDoWork(t *testing.T) {
	ws := NewWorkService(mockTracer{})
	ctx := context.Background()
	start := time.Now()
	ws.DoWork(ctx)
	dur := time.Since(start)
	if dur < 150*time.Millisecond {
		t.Errorf("DoWork should take at least 150ms, got %v", dur)
	}
}
