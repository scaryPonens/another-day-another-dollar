package service

import (
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"
)

func TestExtractOpenAndTargetClose(t *testing.T) {
	open := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	target := open.Add(4 * time.Hour)
	candles := []*domain.Candle{
		{OpenTime: target, Close: 120},
		{OpenTime: open, Close: 100},
		{OpenTime: open.Add(2 * time.Hour), Close: 110},
	}
	openClose, targetClose, ok := extractOpenAndTargetClose(candles, open, target)
	if !ok {
		t.Fatal("expected to find open and target candles")
	}
	if openClose != 100 || targetClose != 120 {
		t.Fatalf("unexpected close values open=%.2f target=%.2f", openClose, targetClose)
	}
}
