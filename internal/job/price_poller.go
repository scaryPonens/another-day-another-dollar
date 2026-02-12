package job

import (
	"context"
	"log"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

// PricePoller runs background goroutines that periodically fetch and store price data.
type PricePoller struct {
	tracer       trace.Tracer
	priceService PriceDataRefresher
	pollInterval time.Duration
}

type PriceDataRefresher interface {
	RefreshPrices(ctx context.Context) error
	RefreshShortCandles(ctx context.Context, symbol string) error
	RefreshLongCandles(ctx context.Context, symbol string) error
}

func NewPricePoller(tracer trace.Tracer, priceService PriceDataRefresher, pollIntervalSecs int) *PricePoller {
	return &PricePoller{
		tracer:       tracer,
		priceService: priceService,
		pollInterval: time.Duration(pollIntervalSecs) * time.Second,
	}
}

// Start launches background polling goroutines. Blocks until ctx is cancelled.
func (p *PricePoller) Start(ctx context.Context) {
	log.Println("Price poller starting...")

	// Tier 1: Current prices every pollInterval (default 60s)
	go p.pollLoop(ctx, "current-prices", p.pollInterval, func(ctx context.Context) error {
		return p.priceService.RefreshPrices(ctx)
	})

	// Tier 2: Short candles (5m, 15m, 1h) — 2 coins every 5 minutes, round-robin
	go p.pollShortCandles(ctx)

	// Tier 3: Long candles (4h, 1d) — 1 coin every 30 minutes, round-robin
	go p.pollLongCandles(ctx)

	<-ctx.Done()
	log.Println("Price poller stopped")
}

func (p *PricePoller) pollLoop(ctx context.Context, name string, interval time.Duration, fn func(context.Context) error) {
	// Run immediately on start
	if err := fn(ctx); err != nil {
		log.Printf("poller %s initial run error: %v", name, err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := fn(ctx); err != nil {
				log.Printf("poller %s error: %v", name, err)
			}
		}
	}
}

func (p *PricePoller) pollShortCandles(ctx context.Context) {
	// Wait a bit before starting to stagger API calls with the price poller
	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second):
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	coinIndex := 0
	coinsPerTick := 2

	// Run immediately
	p.fetchShortBatch(ctx, &coinIndex, coinsPerTick)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.fetchShortBatch(ctx, &coinIndex, coinsPerTick)
		}
	}
}

func (p *PricePoller) fetchShortBatch(ctx context.Context, coinIndex *int, count int) {
	symbols := domain.SupportedSymbols
	for i := 0; i < count; i++ {
		symbol := symbols[*coinIndex%len(symbols)]
		*coinIndex++

		if err := p.priceService.RefreshShortCandles(ctx, symbol); err != nil {
			log.Printf("short candle refresh error for %s: %v", symbol, err)
		}
	}
}

func (p *PricePoller) pollLongCandles(ctx context.Context) {
	// Wait before starting to stagger API calls
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	coinIndex := 0

	// Run immediately
	p.fetchLongBatch(ctx, &coinIndex)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.fetchLongBatch(ctx, &coinIndex)
		}
	}
}

func (p *PricePoller) fetchLongBatch(ctx context.Context, coinIndex *int) {
	symbols := domain.SupportedSymbols
	symbol := symbols[*coinIndex%len(symbols)]
	*coinIndex++

	if err := p.priceService.RefreshLongCandles(ctx, symbol); err != nil {
		log.Printf("long candle refresh error for %s: %v", symbol, err)
	}
}
