package marketintel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/provider"

	"go.opentelemetry.io/otel/trace"
)

type FearGreedReader interface {
	FetchLatest(ctx context.Context) (*provider.FearGreedPoint, error)
}

type RedditReader interface {
	FetchHot(ctx context.Context, subreddit string, limit int) ([]provider.ContentItem, error)
}

type RSSReader interface {
	FetchFeed(ctx context.Context, feedURL string, maxItems int) ([]provider.ContentItem, error)
}

type OnChainReader interface {
	FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*provider.OnChainSnapshot, error)
}

type SignalStore interface {
	InsertSignals(ctx context.Context, signals []domain.Signal) ([]domain.Signal, error)
}

type Store interface {
	UpsertItems(ctx context.Context, items []domain.MarketIntelItem) ([]domain.MarketIntelItem, error)
	UpsertItemSymbols(ctx context.Context, itemID int64, symbols []string) error
	ListUnscoredItems(ctx context.Context, limit int) ([]domain.MarketIntelItem, error)
	UpdateItemSentiment(ctx context.Context, itemID int64, score float64, confidence float64, label string, model string, reason string, scoredAt time.Time) error
	GetSentimentAverages(ctx context.Context, symbol string, from, to time.Time) (map[string]SourceSentimentStats, error)
	UpsertOnChainSnapshot(ctx context.Context, snapshot domain.MarketOnChainSnapshot) (*domain.MarketOnChainSnapshot, error)
	UpsertCompositeSnapshot(ctx context.Context, snapshot domain.MarketCompositeSnapshot) (*domain.MarketCompositeSnapshot, error)
	AttachCompositeSignalID(ctx context.Context, symbol, interval string, openTime time.Time, signalID int64) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type Config struct {
	Intervals         []string
	LongThreshold     float64
	ShortThreshold    float64
	LookbackHours1H   int
	LookbackHours4H   int
	RedditPostLimit   int
	ScoringBatchSize  int
	RetentionDays     int
	EnableOnChain     bool
	OnChainSymbols    []string
	NewsFeeds         []string
	RedditSubs        []string
	NewsFeedItemLimit int
}

type Service struct {
	tracer  trace.Tracer
	repo    Store
	scorer  *Scorer
	signals SignalStore

	fearGreed FearGreedReader
	reddit    RedditReader
	rss       RSSReader
	onchain   map[string]OnChainReader

	cfg Config
}

func NewService(
	tracer trace.Tracer,
	repo Store,
	scorer *Scorer,
	signalStore SignalStore,
	fearGreed FearGreedReader,
	reddit RedditReader,
	rss RSSReader,
	onchain map[string]OnChainReader,
	cfg Config,
) *Service {
	if len(cfg.Intervals) == 0 {
		cfg.Intervals = []string{"1h", "4h"}
	}
	if cfg.LongThreshold <= -1 || cfg.LongThreshold >= 1 {
		cfg.LongThreshold = 0.20
	}
	if cfg.ShortThreshold <= -1 || cfg.ShortThreshold >= 1 {
		cfg.ShortThreshold = -0.20
	}
	if cfg.ShortThreshold > cfg.LongThreshold {
		cfg.ShortThreshold = -0.20
		cfg.LongThreshold = 0.20
	}
	if cfg.LookbackHours1H <= 0 {
		cfg.LookbackHours1H = 12
	}
	if cfg.LookbackHours4H <= 0 {
		cfg.LookbackHours4H = 24
	}
	if cfg.RedditPostLimit <= 0 {
		cfg.RedditPostLimit = 40
	}
	if cfg.ScoringBatchSize <= 0 {
		cfg.ScoringBatchSize = 24
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 90
	}
	if cfg.NewsFeedItemLimit <= 0 {
		cfg.NewsFeedItemLimit = 40
	}
	if scorer == nil {
		scorer = NewScorer(nil, cfg.ScoringBatchSize)
	}
	if onchain == nil {
		onchain = map[string]OnChainReader{}
	}

	return &Service{
		tracer:    tracer,
		repo:      repo,
		scorer:    scorer,
		signals:   signalStore,
		fearGreed: fearGreed,
		reddit:    reddit,
		rss:       rss,
		onchain:   onchain,
		cfg:       cfg,
	}
}

func (s *Service) RunCycle(ctx context.Context, now time.Time) (domain.MarketIntelRunResult, error) {
	_, span := s.tracer.Start(ctx, "market-intel.run-cycle")
	defer span.End()

	if s.repo == nil || s.scorer == nil {
		return domain.MarketIntelRunResult{}, fmt.Errorf("market-intel service dependencies are not initialized")
	}

	now = now.UTC()
	result := domain.MarketIntelRunResult{}
	items := make([]domain.MarketIntelItem, 0, 256)
	symbolSets := make([][]string, 0, 256)
	var fearGreedValue *int

	if s.fearGreed != nil {
		if fg, err := s.fearGreed.FetchLatest(ctx); err != nil {
			result.Errors = append(result.Errors, "fear_greed: "+err.Error())
		} else if fg != nil {
			v := fg.Value
			fearGreedValue = &v
			score := clamp((float64(fg.Value)-50.0)/50.0, -1, 1)
			confidence := clamp(0.4+(0.6*absFloat(score)), 0, 1)
			label := "neutral"
			if score > 0.2 {
				label = "bullish"
			} else if score < -0.2 {
				label = "bearish"
			}
			model := "index:fear_greed_v1"
			reason := strings.TrimSpace(fg.Classification)
			if reason == "" {
				reason = "fear-greed-index"
			}
			meta, _ := json.Marshal(map[string]any{
				"value":             fg.Value,
				"classification":    fg.Classification,
				"time_until_update": fg.TimeUntilUpdateS,
			})
			item := domain.MarketIntelItem{
				Source:              "fear_greed",
				SourceItemID:        fmt.Sprintf("%d", fg.Timestamp.UTC().Unix()),
				Title:               fmt.Sprintf("Fear & Greed: %d (%s)", fg.Value, fg.Classification),
				URL:                 "https://alternative.me/crypto/fear-and-greed-index/",
				Excerpt:             "Crypto market fear and greed index",
				Author:              "alternative.me",
				PublishedAt:         fg.Timestamp.UTC(),
				FetchedAt:           now,
				MetadataJSON:        string(meta),
				SentimentScore:      &score,
				SentimentConfidence: &confidence,
				SentimentLabel:      &label,
				SentimentModel:      &model,
				SentimentReason:     &reason,
				ScoredAt:            &now,
			}
			items = append(items, item)
			symbolSets = append(symbolSets, append([]string(nil), domain.SupportedSymbols...))
		}
	}

	if s.rss != nil {
		for _, feed := range s.cfg.NewsFeeds {
			newsItems, err := s.rss.FetchFeed(ctx, feed, s.cfg.NewsFeedItemLimit)
			if err != nil {
				result.Errors = append(result.Errors, "rss:"+feed+": "+err.Error())
				continue
			}
			for _, row := range newsItems {
				item, symbols := providerContentToItem(now, row)
				items = append(items, item)
				symbolSets = append(symbolSets, symbols)
			}
		}
	}

	if s.reddit != nil {
		for _, subreddit := range s.cfg.RedditSubs {
			posts, err := s.reddit.FetchHot(ctx, subreddit, s.cfg.RedditPostLimit)
			if err != nil {
				result.Errors = append(result.Errors, "reddit:"+subreddit+": "+err.Error())
				continue
			}
			for _, row := range posts {
				item, symbols := providerContentToItem(now, row)
				items = append(items, item)
				symbolSets = append(symbolSets, symbols)
			}
		}
	}

	persisted, err := s.repo.UpsertItems(ctx, items)
	if err != nil {
		return result, err
	}
	result.ItemsIngested += len(persisted)
	for i := range persisted {
		if i >= len(symbolSets) {
			break
		}
		if err := s.repo.UpsertItemSymbols(ctx, persisted[i].ID, symbolSets[i]); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("item_symbols:item=%d: %v", persisted[i].ID, err))
		}
	}

	unscored, err := s.repo.ListUnscoredItems(ctx, maxInt(200, s.cfg.ScoringBatchSize*4))
	if err != nil {
		return result, err
	}
	scored, err := s.scorer.Score(ctx, unscored)
	if err != nil {
		result.Errors = append(result.Errors, "score: "+err.Error())
	}
	for _, row := range scored {
		if err := s.repo.UpdateItemSentiment(ctx, row.ItemID, row.Score, row.Confidence, row.Label, row.Model, row.Reason, now); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("score_update:item=%d: %v", row.ItemID, err))
			continue
		}
		result.ItemsScored++
	}

	onchainBySymbolInterval := make(map[string]domain.MarketOnChainSnapshot)
	if s.cfg.EnableOnChain {
		for _, interval := range s.cfg.Intervals {
			bucket := closedBucket(now, interval)
			for _, symbol := range s.cfg.OnChainSymbols {
				reader := s.onchain[symbol]
				if reader == nil {
					continue
				}
				snapshot, err := reader.FetchSnapshot(ctx, interval, bucket)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("onchain:%s:%s: %v", symbol, interval, err))
					continue
				}
				if snapshot == nil {
					continue
				}
				details, _ := json.Marshal(snapshot.Metrics)
				stored, err := s.repo.UpsertOnChainSnapshot(ctx, domain.MarketOnChainSnapshot{
					Symbol:       snapshot.Symbol,
					Interval:     interval,
					BucketTime:   bucket,
					ProviderKey:  snapshot.ProviderKey,
					OnChainScore: snapshot.Score,
					Confidence:   snapshot.Confidence,
					DetailsJSON:  string(details),
				})
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("onchain_store:%s:%s: %v", symbol, interval, err))
					continue
				}
				key := interval + "|" + symbol
				onchainBySymbolInterval[key] = *stored
				result.OnChainSnapshots++
			}
		}
	}

	for _, interval := range s.cfg.Intervals {
		bucket := closedBucket(now, interval)
		lookbackHours := s.lookbackHours(interval)
		from := bucket.Add(-time.Duration(lookbackHours) * time.Hour)

		for _, symbol := range domain.SupportedSymbols {
			stats, err := s.repo.GetSentimentAverages(ctx, symbol, from, bucket)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("aggregate:%s:%s: %v", symbol, interval, err))
				continue
			}

			input := CompositeInput{
				Interval:       interval,
				LongThreshold:  s.cfg.LongThreshold,
				ShortThreshold: s.cfg.ShortThreshold,
				FearGreedValue: fearGreedValue,
				FearGreed:      componentFromStats(stats["fear_greed"]),
				News:           componentFromStats(stats["news"]),
				Reddit:         componentFromStats(stats["reddit"]),
			}
			if snapshot, ok := onchainBySymbolInterval[interval+"|"+symbol]; ok {
				input.OnChain = CompositeComponent{Score: snapshot.OnChainScore, Confidence: snapshot.Confidence, Available: true}
			}

			computed := BuildComposite(input)
			weightsJSON, _ := json.Marshal(computed.Weights)
			detailsJSON, _ := json.Marshal(map[string]any{
				"model_key":  modelKeyFundSentV1,
				"interval":   interval,
				"score":      computed.Score,
				"confidence": computed.Confidence,
				"details":    computed.DetailsText,
				"lookback_h": lookbackHours,
			})

			snapshot := domain.MarketCompositeSnapshot{
				Symbol:               symbol,
				Interval:             interval,
				OpenTime:             bucket,
				FearGreedValue:       fearGreedValue,
				FearGreedScore:       ptrIfAvailable(input.FearGreed),
				NewsScore:            ptrIfAvailable(input.News),
				RedditScore:          ptrIfAvailable(input.Reddit),
				OnChainScore:         ptrIfAvailable(input.OnChain),
				CompositeScore:       computed.Score,
				Confidence:           computed.Confidence,
				Direction:            computed.Direction,
				Risk:                 computed.Risk,
				ComponentWeightsJSON: string(weightsJSON),
				DetailsJSON:          string(detailsJSON),
			}
			stored, err := s.repo.UpsertCompositeSnapshot(ctx, snapshot)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("composite_store:%s:%s: %v", symbol, interval, err))
				continue
			}
			result.CompositesWritten++

			if computed.Direction == domain.DirectionHold || s.signals == nil {
				continue
			}
			persisted, err := s.signals.InsertSignals(ctx, []domain.Signal{{
				Symbol:    symbol,
				Interval:  interval,
				Indicator: domain.IndicatorFundSentimentComposite,
				Timestamp: bucket,
				Risk:      computed.Risk,
				Direction: computed.Direction,
				Details:   computed.DetailsText,
			}})
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("signal_store:%s:%s: %v", symbol, interval, err))
				continue
			}
			if len(persisted) > 0 && persisted[0].ID > 0 {
				if err := s.repo.AttachCompositeSignalID(ctx, symbol, interval, bucket, persisted[0].ID); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("signal_attach:%s:%s:%d: %v", symbol, interval, persisted[0].ID, err))
				}
			}
			result.SignalsWritten++
			_ = stored
		}
	}

	if s.cfg.RetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -s.cfg.RetentionDays)
		if _, err := s.repo.DeleteOlderThan(ctx, cutoff); err != nil {
			result.Errors = append(result.Errors, "retention: "+err.Error())
		}
	}

	return result, nil
}

func providerContentToItem(now time.Time, row provider.ContentItem) (domain.MarketIntelItem, []string) {
	meta, _ := json.Marshal(row.Metadata)
	symbols := ExtractSymbolsFromContent(row.Source, row.Title, row.Excerpt, row.Metadata)
	return domain.MarketIntelItem{
		Source:       row.Source,
		SourceItemID: row.SourceItemID,
		Title:        strings.TrimSpace(row.Title),
		URL:          strings.TrimSpace(row.URL),
		Excerpt:      strings.TrimSpace(row.Excerpt),
		Author:       strings.TrimSpace(row.Author),
		PublishedAt:  row.PublishedAt.UTC(),
		FetchedAt:    now.UTC(),
		MetadataJSON: string(meta),
	}, symbols
}

func componentFromStats(stat SourceSentimentStats) CompositeComponent {
	if stat.Count <= 0 {
		return CompositeComponent{}
	}
	return CompositeComponent{Score: stat.Score, Confidence: stat.Confidence, Available: true}
}

func ptrIfAvailable(component CompositeComponent) *float64 {
	if !component.Available {
		return nil
	}
	v := component.Score
	return &v
}

func closedBucket(now time.Time, interval string) time.Time {
	now = now.UTC()
	d := time.Hour
	switch interval {
	case "4h":
		d = 4 * time.Hour
	case "1h":
		d = time.Hour
	default:
		d = time.Hour
	}
	return now.Truncate(d).Add(-d)
}

func (s *Service) lookbackHours(interval string) int {
	switch interval {
	case "4h":
		return s.cfg.LookbackHours4H
	default:
		return s.cfg.LookbackHours1H
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
