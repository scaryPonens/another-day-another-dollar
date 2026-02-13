package mcp

import (
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestNormalizeSymbol(t *testing.T) {
	s, err := normalizeSymbol(" btc ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "BTC" {
		t.Fatalf("expected BTC, got %s", s)
	}

	if _, err := normalizeSymbol("fake"); err == nil {
		t.Fatal("expected unsupported symbol error")
	}
}

func TestNormalizeInterval(t *testing.T) {
	iv, err := normalizeInterval("1h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iv != "1h" {
		t.Fatalf("expected 1h, got %s", iv)
	}

	if _, err := normalizeInterval("2h"); err == nil {
		t.Fatal("expected unsupported interval error")
	}
}

func TestNormalizeSignalFilter(t *testing.T) {
	r := 3
	filter, err := normalizeSignalFilter(signalsListInput{
		Symbol:    "btc",
		Risk:      &r,
		Indicator: "MACD",
		Limit:     999,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filter.Symbol != "BTC" {
		t.Fatalf("expected symbol BTC, got %s", filter.Symbol)
	}
	if filter.Indicator != domain.IndicatorMACD {
		t.Fatalf("expected macd indicator, got %s", filter.Indicator)
	}
	if filter.Risk == nil || *filter.Risk != domain.RiskLevel3 {
		t.Fatalf("unexpected risk %+v", filter.Risk)
	}
	if filter.Limit != maxSignalLimit {
		t.Fatalf("expected capped signal limit %d, got %d", maxSignalLimit, filter.Limit)
	}
}

func TestNormalizeGenerateIntervals(t *testing.T) {
	ivs, err := normalizeGenerateIntervals(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ivs) != len(domain.SupportedIntervals) {
		t.Fatalf("expected all intervals, got %d", len(ivs))
	}

	ivs, err = normalizeGenerateIntervals([]string{"1h", "1h", "5m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ivs) != 2 {
		t.Fatalf("expected deduped intervals, got %d", len(ivs))
	}
}

func TestNormalizeIndicatorML(t *testing.T) {
	got, err := normalizeIndicator("ml_logreg_up4h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domain.IndicatorMLLogRegUp4H {
		t.Fatalf("expected ml indicator, got %s", got)
	}

	got, err = normalizeIndicator("fund_sentiment_composite")
	if err != nil {
		t.Fatalf("unexpected error for phase7 indicator: %v", err)
	}
	if got != domain.IndicatorFundSentimentComposite {
		t.Fatalf("expected phase7 indicator, got %s", got)
	}
}
