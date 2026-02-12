package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

const coingeckoBaseURL = "https://api.coingecko.com/api/v3"

// CoinGeckoProvider fetches price and OHLC data from the CoinGecko free API.
type CoinGeckoProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
	limiter *RateLimiter
}

// NewCoinGeckoProvider creates a new provider with built-in rate limiting.
// Rate limited to 8 requests per minute (one token every 7.5 seconds).
func NewCoinGeckoProvider(tracer trace.Tracer) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: coingeckoBaseURL,
		tracer:  tracer,
		limiter: NewRateLimiter(8, 7500*time.Millisecond),
	}
}

// FetchPrices fetches current prices for all supported assets in a single API call.
func (p *CoinGeckoProvider) FetchPrices(ctx context.Context) (map[string]*domain.PriceSnapshot, error) {
	_, span := p.tracer.Start(ctx, "coingecko.fetch-prices")
	defer span.End()

	ids := make([]string, 0, len(domain.CoinGeckoID))
	for _, id := range domain.CoinGeckoID {
		ids = append(ids, id)
	}

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd&include_24hr_vol=true&include_24hr_change=true",
		p.baseURL, strings.Join(ids, ","))

	body, err := p.doRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch prices: %w", err)
	}

	// Response shape: {"bitcoin": {"usd": 97000, "usd_24h_vol": 45000000000, "usd_24h_change": 2.34}, ...}
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse prices: %w", err)
	}

	now := time.Now().Unix()
	result := make(map[string]*domain.PriceSnapshot, len(raw))
	for cgID, data := range raw {
		symbol, ok := domain.CoinGeckoIDToSymbol[cgID]
		if !ok {
			continue
		}
		result[symbol] = &domain.PriceSnapshot{
			Symbol:          symbol,
			PriceUSD:        data["usd"],
			Volume24h:       data["usd_24h_vol"],
			Change24hPct:    data["usd_24h_change"],
			LastUpdatedUnix: now,
		}
	}

	return result, nil
}

// FetchMarketChart fetches market_chart data and constructs candles for the given intervals.
// days=1 gives ~5min granularity (for 5m, 15m, 1h candles).
// days=30 gives ~1h granularity (for 4h, 1d candles).
func (p *CoinGeckoProvider) FetchMarketChart(ctx context.Context, symbol string, days int, intervals []string) ([]*domain.Candle, error) {
	_, span := p.tracer.Start(ctx, "coingecko.fetch-market-chart")
	defer span.End()

	cgID, ok := domain.CoinGeckoID[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported symbol: %s", symbol)
	}

	url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=usd&days=%d",
		p.baseURL, cgID, days)

	body, err := p.doRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch market chart for %s: %w", symbol, err)
	}

	var raw struct {
		Prices       [][]float64 `json:"prices"`
		TotalVolumes [][]float64 `json:"total_volumes"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse market chart for %s: %w", symbol, err)
	}

	var allCandles []*domain.Candle
	for _, interval := range intervals {
		candles := buildCandlesFromMarketChart(symbol, interval, raw.Prices, raw.TotalVolumes)
		allCandles = append(allCandles, candles...)
	}

	return allCandles, nil
}

func (p *CoinGeckoProvider) doRequest(ctx context.Context, url string) ([]byte, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("coingecko API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

type volumePoint struct {
	ts  int64
	vol float64
}

// buildCandlesFromMarketChart constructs candles of the given interval
// from raw market_chart price/volume arrays.
func buildCandlesFromMarketChart(symbol, interval string, prices, volumes [][]float64) []*domain.Candle {
	if len(prices) == 0 {
		return nil
	}

	intervalDuration := intervalToDuration(interval)
	if intervalDuration == 0 {
		return nil
	}

	// Build volume lookup by timestamp for closest-match volume assignment
	volPoints := make([]volumePoint, 0, len(volumes))
	for _, v := range volumes {
		if len(v) >= 2 {
			volPoints = append(volPoints, volumePoint{ts: int64(v[0]), vol: v[1]})
		}
	}

	// Sort prices by timestamp
	sort.Slice(prices, func(i, j int) bool {
		return prices[i][0] < prices[j][0]
	})

	// Bucket prices into candle windows
	type bucket struct {
		open      float64
		high      float64
		low       float64
		close     float64
		openTime  time.Time
		lastVolTS int64
	}

	buckets := make(map[int64]*bucket)

	for _, pt := range prices {
		if len(pt) < 2 {
			continue
		}
		tsMs := int64(pt[0])
		price := pt[1]
		t := time.UnixMilli(tsMs)

		// Floor to interval boundary
		bucketTS := t.Truncate(intervalDuration).UnixMilli()

		b, exists := buckets[bucketTS]
		if !exists {
			b = &bucket{
				open:     price,
				high:     price,
				low:      price,
				close:    price,
				openTime: time.UnixMilli(bucketTS),
			}
			buckets[bucketTS] = b
		} else {
			b.high = math.Max(b.high, price)
			b.low = math.Min(b.low, price)
			b.close = price // last price in the bucket becomes the close
		}
	}

	// Build sorted candle list
	sortedKeys := make([]int64, 0, len(buckets))
	for k := range buckets {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })

	// Assign volume: find the closest volume point for each bucket
	candles := make([]*domain.Candle, 0, len(sortedKeys))
	for _, k := range sortedKeys {
		b := buckets[k]
		vol := findClosestVolume(volPoints, k+int64(intervalDuration/time.Millisecond))
		candles = append(candles, &domain.Candle{
			Symbol:   symbol,
			Interval: interval,
			OpenTime: b.openTime.UTC(),
			Open:     b.open,
			High:     b.high,
			Low:      b.low,
			Close:    b.close,
			Volume:   vol,
		})
	}

	return candles
}

func findClosestVolume(volumes []volumePoint, targetMs int64) float64 {
	if len(volumes) == 0 {
		return 0
	}
	closest := volumes[0]
	minDiff := int64(math.MaxInt64)
	for _, v := range volumes {
		diff := v.ts - targetMs
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closest = v
		}
	}
	return closest.vol
}

func intervalToDuration(interval string) time.Duration {
	switch interval {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 0
	}
}
