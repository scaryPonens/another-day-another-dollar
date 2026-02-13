package job

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

func TestMarketIntelJobRunsAtLeastOnce(t *testing.T) {
	var calls int32
	runner := &marketIntelRunnerTestStub{calls: &calls}
	job := NewMarketIntelJob(trace.NewNoopTracerProvider().Tracer("test"), runner, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		job.Start(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("expected at least one market intel run")
	}
}

type marketIntelRunnerTestStub struct {
	calls *int32
}

func (s *marketIntelRunnerTestStub) RunMarketIntel(ctx context.Context) (domain.MarketIntelRunResult, error) {
	atomic.AddInt32(s.calls, 1)
	return domain.MarketIntelRunResult{}, nil
}
