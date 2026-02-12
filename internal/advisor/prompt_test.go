package advisor

import (
	"strings"
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestBuildSystemPromptContainsPhilosophy(t *testing.T) {
	prompt := BuildSystemPrompt("some context")
	if !strings.Contains(prompt, "crypto trading advisor") {
		t.Fatal("expected trading philosophy in prompt")
	}
	if !strings.Contains(prompt, "Risk Framework") {
		t.Fatal("expected risk framework in prompt")
	}
	if !strings.Contains(prompt, "LIVE MARKET DATA") {
		t.Fatal("expected market data header in prompt")
	}
	if !strings.Contains(prompt, "some context") {
		t.Fatal("expected market context in prompt")
	}
}

func TestFormatMarketContextWithPricesAndSignals(t *testing.T) {
	prices := []*domain.PriceSnapshot{
		{Symbol: "BTC", PriceUSD: 50000, Change24hPct: 2.5, Volume24h: 1e9},
	}
	signals := []domain.Signal{
		{Symbol: "BTC", Interval: "1h", Indicator: "rsi", Direction: domain.DirectionLong, Risk: 2, Details: "oversold"},
	}

	ctx := FormatMarketContext(prices, signals)
	if !strings.Contains(ctx, "BTC: $50000.00") {
		t.Fatal("expected BTC price in context")
	}
	if !strings.Contains(ctx, "RSI") {
		t.Fatal("expected RSI indicator in context")
	}
	if !strings.Contains(ctx, "LONG") {
		t.Fatal("expected LONG direction in context")
	}
	if !strings.Contains(ctx, "risk=2") {
		t.Fatal("expected risk level in context")
	}
}

func TestFormatMarketContextEmpty(t *testing.T) {
	ctx := FormatMarketContext(nil, nil)
	if ctx != "No market data currently available." {
		t.Fatalf("expected fallback text, got: %s", ctx)
	}
}

func TestFormatMarketContextPricesOnly(t *testing.T) {
	prices := []*domain.PriceSnapshot{
		{Symbol: "ETH", PriceUSD: 3000, Change24hPct: -1.2, Volume24h: 5e8},
	}
	ctx := FormatMarketContext(prices, nil)
	if !strings.Contains(ctx, "ETH: $3000.00") {
		t.Fatal("expected ETH price")
	}
	if strings.Contains(ctx, "Active Signals") {
		t.Fatal("should not contain signals section when no signals")
	}
}
