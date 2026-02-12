package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

const priceCacheTTL = 90 * time.Second

// PriceService orchestrates price data fetching, caching, and retrieval.
type PriceProvider interface {
	FetchPrices(ctx context.Context) (map[string]*domain.PriceSnapshot, error)
	FetchMarketChart(ctx context.Context, symbol string, days int, intervals []string) ([]*domain.Candle, error)
}

type CandleRepository interface {
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error)
	UpsertCandles(ctx context.Context, candles []*domain.Candle) error
}

type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
}

type PriceService struct {
	tracer   trace.Tracer
	provider PriceProvider
	repo     CandleRepository
	redis    RedisClient
}

func NewPriceService(
	tracer trace.Tracer,
	provider PriceProvider,
	repo CandleRepository,
	redisClient RedisClient,
) *PriceService {
	return &PriceService{
		tracer:   tracer,
		provider: provider,
		repo:     repo,
		redis:    redisClient,
	}
}

// GetCurrentPrice returns the latest cached price for a symbol.
// Falls back to a live API call if cache is empty/expired.
func (s *PriceService) GetCurrentPrice(ctx context.Context, symbol string) (*domain.PriceSnapshot, error) {
	_, span := s.tracer.Start(ctx, "price-service.get-current-price")
	defer span.End()

	if _, ok := domain.CoinGeckoID[symbol]; !ok {
		return nil, fmt.Errorf("unsupported symbol: %s", symbol)
	}

	// Try Redis cache
	if s.redis != nil {
		cached, err := s.getPriceCache(ctx, symbol)
		if err != nil {
			log.Printf("redis cache read error: %v", err)
		}
		if cached != nil {
			return cached, nil
		}
	}

	// Cache miss: fetch all prices (single batched API call), cache them
	prices, err := s.provider.FetchPrices(ctx)
	if err != nil {
		return nil, err
	}

	for _, snap := range prices {
		if s.redis != nil {
			_ = s.setPriceCache(ctx, snap)
		}
	}

	snap, ok := prices[symbol]
	if !ok {
		return nil, fmt.Errorf("price not available for %s", symbol)
	}
	return snap, nil
}

// GetCurrentPrices returns latest cached prices for all supported symbols.
func (s *PriceService) GetCurrentPrices(ctx context.Context) ([]*domain.PriceSnapshot, error) {
	_, span := s.tracer.Start(ctx, "price-service.get-current-prices")
	defer span.End()

	var snapshots []*domain.PriceSnapshot
	var missing []string

	for _, symbol := range domain.SupportedSymbols {
		if s.redis != nil {
			cached, _ := s.getPriceCache(ctx, symbol)
			if cached != nil {
				snapshots = append(snapshots, cached)
				continue
			}
		}
		missing = append(missing, symbol)
	}

	if len(missing) > 0 {
		prices, err := s.provider.FetchPrices(ctx)
		if err != nil {
			return snapshots, err
		}
		for _, snap := range prices {
			if s.redis != nil {
				_ = s.setPriceCache(ctx, snap)
			}
			snapshots = append(snapshots, snap)
		}
	}

	return snapshots, nil
}

// GetCandles returns historical candles for a symbol and interval from Postgres.
func (s *PriceService) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]*domain.Candle, error) {
	return s.repo.GetCandles(ctx, symbol, interval, limit)
}

// RefreshPrices fetches latest prices from CoinGecko and caches in Redis.
func (s *PriceService) RefreshPrices(ctx context.Context) error {
	_, span := s.tracer.Start(ctx, "price-service.refresh-prices")
	defer span.End()

	prices, err := s.provider.FetchPrices(ctx)
	if err != nil {
		return err
	}

	for _, snap := range prices {
		if s.redis != nil {
			if err := s.setPriceCache(ctx, snap); err != nil {
				log.Printf("redis cache write error for %s: %v", snap.Symbol, err)
			}
		}
	}

	log.Printf("Refreshed prices for %d assets", len(prices))
	return nil
}

// RefreshShortCandles fetches market_chart data (days=1) and stores 5m, 15m, 1h candles.
func (s *PriceService) RefreshShortCandles(ctx context.Context, symbol string) error {
	_, span := s.tracer.Start(ctx, "price-service.refresh-short-candles")
	defer span.End()

	candles, err := s.provider.FetchMarketChart(ctx, symbol, 1, []string{"5m", "15m", "1h"})
	if err != nil {
		return err
	}

	if err := s.repo.UpsertCandles(ctx, candles); err != nil {
		return fmt.Errorf("upsert short candles for %s: %w", symbol, err)
	}

	log.Printf("Refreshed short candles for %s (%d candles)", symbol, len(candles))
	return nil
}

// RefreshLongCandles fetches market_chart data (days=30) and stores 4h, 1d candles.
func (s *PriceService) RefreshLongCandles(ctx context.Context, symbol string) error {
	_, span := s.tracer.Start(ctx, "price-service.refresh-long-candles")
	defer span.End()

	candles, err := s.provider.FetchMarketChart(ctx, symbol, 30, []string{"4h", "1d"})
	if err != nil {
		return err
	}

	if err := s.repo.UpsertCandles(ctx, candles); err != nil {
		return fmt.Errorf("upsert long candles for %s: %w", symbol, err)
	}

	log.Printf("Refreshed long candles for %s (%d candles)", symbol, len(candles))
	return nil
}

func (s *PriceService) setPriceCache(ctx context.Context, snapshot *domain.PriceSnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, "price:"+snapshot.Symbol, data, priceCacheTTL).Err()
}

func (s *PriceService) getPriceCache(ctx context.Context, symbol string) (*domain.PriceSnapshot, error) {
	data, err := s.redis.Get(ctx, "price:"+symbol).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snapshot domain.PriceSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}
