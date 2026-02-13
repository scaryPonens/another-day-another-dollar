package marketintel

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/trace"
)

type pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type Repository struct {
	pool   pool
	tracer trace.Tracer
}

type SourceSentimentStats struct {
	Score      float64
	Confidence float64
	Count      int
}

func NewRepository(pool pool, tracer trace.Tracer) *Repository {
	return &Repository{pool: pool, tracer: tracer}
}

func (r *Repository) UpsertItems(ctx context.Context, items []domain.MarketIntelItem) ([]domain.MarketIntelItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	_, span := r.tracer.Start(ctx, "market-intel-repo.upsert-items")
	defer span.End()

	batch := &pgx.Batch{}
	for _, item := range items {
		metadata := ensureJSON(item.MetadataJSON)
		batch.Queue(`
INSERT INTO market_intel_items (
    source, source_item_id, title, url, excerpt, author,
    published_at, fetched_at, metadata_json,
    sentiment_score, sentiment_confidence, sentiment_label, sentiment_model, sentiment_reason, scored_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13, $14, $15
)
ON CONFLICT (source, source_item_id) DO UPDATE SET
    title = EXCLUDED.title,
    url = EXCLUDED.url,
    excerpt = EXCLUDED.excerpt,
    author = EXCLUDED.author,
    published_at = EXCLUDED.published_at,
    fetched_at = EXCLUDED.fetched_at,
    metadata_json = EXCLUDED.metadata_json,
    sentiment_score = COALESCE(EXCLUDED.sentiment_score, market_intel_items.sentiment_score),
    sentiment_confidence = COALESCE(EXCLUDED.sentiment_confidence, market_intel_items.sentiment_confidence),
    sentiment_label = COALESCE(EXCLUDED.sentiment_label, market_intel_items.sentiment_label),
    sentiment_model = COALESCE(EXCLUDED.sentiment_model, market_intel_items.sentiment_model),
    sentiment_reason = COALESCE(EXCLUDED.sentiment_reason, market_intel_items.sentiment_reason),
    scored_at = COALESCE(EXCLUDED.scored_at, market_intel_items.scored_at),
    updated_at = NOW()
RETURNING id, source, source_item_id, title, url, excerpt, author,
          published_at, fetched_at, metadata_json,
          sentiment_score, sentiment_confidence, sentiment_label, sentiment_model, sentiment_reason, scored_at,
          created_at, updated_at, '{}'::text[]`,
			item.Source,
			item.SourceItemID,
			item.Title,
			item.URL,
			item.Excerpt,
			item.Author,
			item.PublishedAt.UTC(),
			nullIfZeroTime(item.FetchedAt),
			metadata,
			nullFloat(item.SentimentScore),
			nullFloat(item.SentimentConfidence),
			nullString(item.SentimentLabel),
			nullString(item.SentimentModel),
			nullString(item.SentimentReason),
			nullTime(item.ScoredAt),
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	out := make([]domain.MarketIntelItem, 0, len(items))
	for range items {
		item, err := scanMarketIntelItemRow(br.QueryRow())
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *Repository) UpsertItemSymbols(ctx context.Context, itemID int64, symbols []string) error {
	_, span := r.tracer.Start(ctx, "market-intel-repo.upsert-item-symbols")
	defer span.End()

	if itemID <= 0 || len(symbols) == 0 {
		return nil
	}
	unique := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbol = normalizeSymbol(symbol)
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		unique = append(unique, symbol)
	}
	if len(unique) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, symbol := range unique {
		batch.Queue(`
INSERT INTO market_intel_item_symbols (item_id, symbol)
VALUES ($1, $2)
ON CONFLICT (item_id, symbol) DO NOTHING`, itemID, symbol)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range unique {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListUnscoredItems(ctx context.Context, limit int) ([]domain.MarketIntelItem, error) {
	_, span := r.tracer.Start(ctx, "market-intel-repo.list-unscored-items")
	defer span.End()

	if limit <= 0 {
		limit = 200
	}

	rows, err := r.pool.Query(ctx, `
SELECT i.id, i.source, i.source_item_id, i.title, i.url, i.excerpt, i.author,
       i.published_at, i.fetched_at, i.metadata_json,
       i.sentiment_score, i.sentiment_confidence, i.sentiment_label, i.sentiment_model, i.sentiment_reason,
       i.scored_at, i.created_at, i.updated_at,
       COALESCE(array_agg(ms.symbol) FILTER (WHERE ms.symbol IS NOT NULL), '{}'::text[])
FROM market_intel_items i
LEFT JOIN market_intel_item_symbols ms ON ms.item_id = i.id
WHERE i.scored_at IS NULL
GROUP BY i.id
ORDER BY i.published_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.MarketIntelItem, 0, limit)
	for rows.Next() {
		item, err := scanMarketIntelItemRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateItemSentiment(
	ctx context.Context,
	itemID int64,
	score float64,
	confidence float64,
	label string,
	model string,
	reason string,
	scoredAt time.Time,
) error {
	_, span := r.tracer.Start(ctx, "market-intel-repo.update-item-sentiment")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `
UPDATE market_intel_items
SET sentiment_score = $2,
    sentiment_confidence = $3,
    sentiment_label = $4,
    sentiment_model = $5,
    sentiment_reason = $6,
    scored_at = $7,
    updated_at = NOW()
WHERE id = $1`, itemID, score, confidence, label, model, reason, scoredAt.UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetSentimentAverages(ctx context.Context, symbol string, from, to time.Time) (map[string]SourceSentimentStats, error) {
	_, span := r.tracer.Start(ctx, "market-intel-repo.get-sentiment-averages")
	defer span.End()

	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return map[string]SourceSentimentStats{}, nil
	}

	rows, err := r.pool.Query(ctx, `
SELECT i.source,
       AVG(i.sentiment_score) AS avg_score,
       AVG(i.sentiment_confidence) AS avg_conf,
       COUNT(*)::INT AS n
FROM market_intel_items i
JOIN market_intel_item_symbols s ON s.item_id = i.id
WHERE s.symbol = $1
  AND i.scored_at IS NOT NULL
  AND i.published_at >= $2
  AND i.published_at <= $3
GROUP BY i.source`, symbol, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]SourceSentimentStats)
	for rows.Next() {
		var source string
		var score float64
		var confidence float64
		var count int
		if err := rows.Scan(&source, &score, &confidence, &count); err != nil {
			return nil, err
		}
		out[source] = SourceSentimentStats{Score: score, Confidence: confidence, Count: count}
	}
	return out, rows.Err()
}

func (r *Repository) UpsertOnChainSnapshot(ctx context.Context, snapshot domain.MarketOnChainSnapshot) (*domain.MarketOnChainSnapshot, error) {
	_, span := r.tracer.Start(ctx, "market-intel-repo.upsert-onchain-snapshot")
	defer span.End()

	var out domain.MarketOnChainSnapshot
	err := r.pool.QueryRow(ctx, `
INSERT INTO market_onchain_snapshots (
    symbol, interval, bucket_time, provider_key,
    onchain_score, confidence, details_json
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7
)
ON CONFLICT (symbol, interval, bucket_time, provider_key) DO UPDATE SET
    onchain_score = EXCLUDED.onchain_score,
    confidence = EXCLUDED.confidence,
    details_json = EXCLUDED.details_json,
    created_at = NOW()
RETURNING symbol, interval, bucket_time, provider_key,
          onchain_score, confidence, details_json, created_at`,
		normalizeSymbol(snapshot.Symbol), snapshot.Interval, snapshot.BucketTime.UTC(), snapshot.ProviderKey,
		snapshot.OnChainScore, snapshot.Confidence, ensureJSON(snapshot.DetailsJSON),
	).Scan(
		&out.Symbol,
		&out.Interval,
		&out.BucketTime,
		&out.ProviderKey,
		&out.OnChainScore,
		&out.Confidence,
		&out.DetailsJSON,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	out.BucketTime = out.BucketTime.UTC()
	out.CreatedAt = out.CreatedAt.UTC()
	return &out, nil
}

func (r *Repository) UpsertCompositeSnapshot(ctx context.Context, snapshot domain.MarketCompositeSnapshot) (*domain.MarketCompositeSnapshot, error) {
	_, span := r.tracer.Start(ctx, "market-intel-repo.upsert-composite-snapshot")
	defer span.End()

	var out domain.MarketCompositeSnapshot
	err := r.pool.QueryRow(ctx, `
INSERT INTO market_composite_snapshots (
    symbol, interval, open_time,
    fear_greed_value, fear_greed_score, news_score, reddit_score, onchain_score,
    composite_score, confidence, direction, risk,
    component_weights_json, details_json, signal_id
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8,
    $9, $10, $11, $12,
    $13, $14, $15
)
ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
    fear_greed_value = EXCLUDED.fear_greed_value,
    fear_greed_score = EXCLUDED.fear_greed_score,
    news_score = EXCLUDED.news_score,
    reddit_score = EXCLUDED.reddit_score,
    onchain_score = EXCLUDED.onchain_score,
    composite_score = EXCLUDED.composite_score,
    confidence = EXCLUDED.confidence,
    direction = EXCLUDED.direction,
    risk = EXCLUDED.risk,
    component_weights_json = EXCLUDED.component_weights_json,
    details_json = EXCLUDED.details_json,
    updated_at = NOW()
RETURNING symbol, interval, open_time,
          fear_greed_value, fear_greed_score, news_score, reddit_score, onchain_score,
          composite_score, confidence, direction, risk,
          component_weights_json, details_json, signal_id, created_at, updated_at`,
		normalizeSymbol(snapshot.Symbol), snapshot.Interval, snapshot.OpenTime.UTC(),
		nullInt(snapshot.FearGreedValue),
		nullFloat(snapshot.FearGreedScore),
		nullFloat(snapshot.NewsScore),
		nullFloat(snapshot.RedditScore),
		nullFloat(snapshot.OnChainScore),
		snapshot.CompositeScore,
		snapshot.Confidence,
		string(snapshot.Direction),
		int16(snapshot.Risk),
		ensureJSON(snapshot.ComponentWeightsJSON),
		ensureJSON(snapshot.DetailsJSON),
		snapshot.SignalID,
	).Scan(
		&out.Symbol,
		&out.Interval,
		&out.OpenTime,
		&out.FearGreedValue,
		&out.FearGreedScore,
		&out.NewsScore,
		&out.RedditScore,
		&out.OnChainScore,
		&out.CompositeScore,
		&out.Confidence,
		&out.Direction,
		&out.Risk,
		&out.ComponentWeightsJSON,
		&out.DetailsJSON,
		&out.SignalID,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	out.OpenTime = out.OpenTime.UTC()
	out.CreatedAt = out.CreatedAt.UTC()
	out.UpdatedAt = out.UpdatedAt.UTC()
	return &out, nil
}

func (r *Repository) AttachCompositeSignalID(ctx context.Context, symbol, interval string, openTime time.Time, signalID int64) error {
	_, span := r.tracer.Start(ctx, "market-intel-repo.attach-composite-signal-id")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `
UPDATE market_composite_snapshots
SET signal_id = $4, updated_at = NOW()
WHERE symbol = $1 AND interval = $2 AND open_time = $3`, normalizeSymbol(symbol), interval, openTime.UTC(), signalID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	_, span := r.tracer.Start(ctx, "market-intel-repo.delete-older-than")
	defer span.End()

	total := int64(0)
	queries := []string{
		`DELETE FROM market_intel_items WHERE published_at < $1`,
		`DELETE FROM market_onchain_snapshots WHERE bucket_time < $1`,
		`DELETE FROM market_composite_snapshots WHERE open_time < $1`,
	}
	for _, q := range queries {
		tag, err := r.pool.Exec(ctx, q, cutoff.UTC())
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
	}
	return total, nil
}

func scanMarketIntelItemRow(s interface{ Scan(dest ...any) error }) (domain.MarketIntelItem, error) {
	var out domain.MarketIntelItem
	var score pgtype.Float8
	var conf pgtype.Float8
	var label pgtype.Text
	var model pgtype.Text
	var reason pgtype.Text
	var scored pgtype.Timestamptz
	var symbols []string

	if err := s.Scan(
		&out.ID,
		&out.Source,
		&out.SourceItemID,
		&out.Title,
		&out.URL,
		&out.Excerpt,
		&out.Author,
		&out.PublishedAt,
		&out.FetchedAt,
		&out.MetadataJSON,
		&score,
		&conf,
		&label,
		&model,
		&reason,
		&scored,
		&out.CreatedAt,
		&out.UpdatedAt,
		&symbols,
	); err != nil {
		return domain.MarketIntelItem{}, err
	}

	out.PublishedAt = out.PublishedAt.UTC()
	out.FetchedAt = out.FetchedAt.UTC()
	out.CreatedAt = out.CreatedAt.UTC()
	out.UpdatedAt = out.UpdatedAt.UTC()
	if score.Valid {
		v := score.Float64
		out.SentimentScore = &v
	}
	if conf.Valid {
		v := conf.Float64
		out.SentimentConfidence = &v
	}
	if label.Valid {
		v := label.String
		out.SentimentLabel = &v
	}
	if model.Valid {
		v := model.String
		out.SentimentModel = &v
	}
	if reason.Valid {
		v := reason.String
		out.SentimentReason = &v
	}
	if scored.Valid {
		v := scored.Time.UTC()
		out.ScoredAt = &v
	}
	out.Symbols = normalizeSymbolList(symbols)
	return out, nil
}

func normalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return ""
	}
	if _, ok := domain.CoinGeckoID[symbol]; !ok {
		return ""
	}
	return symbol
}

func normalizeSymbolList(symbols []string) []string {
	if len(symbols) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(symbols))
	out := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		norm := normalizeSymbol(symbol)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	sort.Strings(out)
	return out
}

func ensureJSON(raw string) string {
	if raw == "" {
		return "{}"
	}
	if json.Valid([]byte(raw)) {
		return raw
	}
	encoded, err := json.Marshal(map[string]string{"raw": raw})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func nullFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullString(v *string) any {
	if v == nil {
		return nil
	}
	if *v == "" {
		return nil
	}
	return *v
}

func nullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullTime(v *time.Time) any {
	if v == nil || v.IsZero() {
		return nil
	}
	return v.UTC()
}

func nullIfZeroTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return v.UTC()
}
