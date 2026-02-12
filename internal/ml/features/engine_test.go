package features

import (
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"
)

func TestEngineBuildRowsDeterministic(t *testing.T) {
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	engine := NewEngine(func() time.Time { return now })
	candles := makeCandles(48)

	rowsA := engine.BuildRows(candles, 4)
	rowsB := engine.BuildRows(candles, 4)
	if len(rowsA) == 0 {
		t.Fatal("expected non-empty feature rows")
	}
	if len(rowsA) != len(rowsB) {
		t.Fatalf("expected deterministic row count, got %d vs %d", len(rowsA), len(rowsB))
	}
	if rowsA[0].Ret1H != rowsB[0].Ret1H || rowsA[0].RSI14 != rowsB[0].RSI14 {
		t.Fatalf("expected deterministic features, got %+v vs %+v", rowsA[0], rowsB[0])
	}
	if !rowsA[0].CreatedAt.Equal(now) {
		t.Fatalf("expected created_at from injected clock, got %s", rowsA[0].CreatedAt)
	}

	hasLabeled := false
	hasUnlabeled := false
	for _, row := range rowsA {
		if row.TargetUp4H != nil {
			hasLabeled = true
		} else {
			hasUnlabeled = true
		}
	}
	if !hasLabeled || !hasUnlabeled {
		t.Fatalf("expected both labeled and unlabeled rows, got labeled=%v unlabeled=%v", hasLabeled, hasUnlabeled)
	}
}

func makeCandles(n int) []*domain.Candle {
	out := make([]*domain.Candle, 0, n)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < n; i++ {
		price += 0.8
		out = append(out, &domain.Candle{
			Symbol:   "BTC",
			Interval: "1h",
			OpenTime: start.Add(time.Duration(i) * time.Hour),
			Open:     price - 0.2,
			High:     price + 0.4,
			Low:      price - 0.6,
			Close:    price,
			Volume:   1000 + float64(i*10),
		})
	}
	return out
}
