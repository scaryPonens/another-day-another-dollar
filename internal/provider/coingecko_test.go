package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestBuildCandlesFromMarketChart(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := [][]float64{
		{float64(base.UnixMilli()), 10},
		{float64(base.Add(2 * time.Minute).UnixMilli()), 12},
		{float64(base.Add(6 * time.Minute).UnixMilli()), 8},
		{float64(base.Add(8 * time.Minute).UnixMilli()), 9},
	}
	volumes := [][]float64{
		{float64(base.Add(5 * time.Minute).UnixMilli()), 100},
		{float64(base.Add(10 * time.Minute).UnixMilli()), 200},
	}

	candles := buildCandlesFromMarketChart("BTC", "5m", prices, volumes)
	if len(candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(candles))
	}

	first := candles[0]
	if first.Open != 10 || first.High != 12 || first.Low != 10 || first.Close != 12 {
		t.Fatalf("unexpected first candle: %+v", first)
	}
	if first.Volume != 100 {
		t.Fatalf("expected volume 100, got %f", first.Volume)
	}

	second := candles[1]
	if !second.OpenTime.Equal(base.Add(5 * time.Minute)) {
		t.Fatalf("unexpected open time: %v", second.OpenTime)
	}
	if second.Open != 8 || second.Close != 9 {
		t.Fatalf("unexpected second candle: %+v", second)
	}
}

func TestFindClosestVolume(t *testing.T) {
	volumes := []volumePoint{
		{ts: 1000, vol: 1},
		{ts: 1500, vol: 5},
		{ts: 2000, vol: 10},
	}
	vol := findClosestVolume(volumes, 1600)
	if vol != 5 {
		t.Fatalf("expected volume 5, got %f", vol)
	}
}

func TestIntervalToDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
		"1d":  24 * time.Hour,
		"bad": 0,
	}
	for interval, expected := range tests {
		if got := intervalToDuration(interval); got != expected {
			t.Fatalf("%s expected %v, got %v", interval, expected, got)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCoinGeckoProviderFetchPrices(t *testing.T) {
	t.Parallel()

	provider := NewCoinGeckoProvider(trace.NewNoopTracerProvider().Tracer("test"))
	provider.baseURL = "http://example"
	provider.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/simple/price") {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			resp := map[string]map[string]float64{
				"bitcoin": {"usd": 100, "usd_24h_vol": 10, "usd_24h_change": 1.5},
			}
			data, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	provider.limiter = NewRateLimiter(10, time.Millisecond)

	result, err := provider.FetchPrices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap, ok := result["BTC"]
	if !ok || snap.PriceUSD != 100 {
		t.Fatalf("expected BTC snapshot, got %+v", snap)
	}
	if snap.Volume24h != 10 || snap.Change24hPct != 1.5 {
		t.Fatalf("unexpected snapshot values: %+v", snap)
	}
}

func TestCoinGeckoProviderFetchMarketChart(t *testing.T) {
	t.Parallel()

	provider := NewCoinGeckoProvider(trace.NewNoopTracerProvider().Tracer("test"))
	provider.baseURL = "http://example"
	provider.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/coins/bitcoin/market_chart") {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			resp := map[string]interface{}{
				"prices": [][]float64{
					{float64(time.Now().Add(-10 * time.Minute).UnixMilli()), 10},
					{float64(time.Now().UnixMilli()), 12},
				},
				"total_volumes": [][]float64{
					{float64(time.Now().UnixMilli()), 100},
				},
			}
			data, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	provider.limiter = NewRateLimiter(10, time.Millisecond)

	candles, err := provider.FetchMarketChart(context.Background(), "BTC", 1, []string{"5m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) == 0 {
		t.Fatalf("expected candles, got none")
	}
	if candles[0].Symbol != "BTC" {
		t.Fatalf("expected BTC candles, got %+v", candles[0])
	}
}
