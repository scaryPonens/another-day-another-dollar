package marketintel

import (
	"context"
	"errors"
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestScorerHeuristicFallback(t *testing.T) {
	scorer := NewScorer(nil, 10)
	items := []domain.MarketIntelItem{{ID: 1, Title: "Bitcoin breakout", Excerpt: "bull trend"}}

	out, err := scorer.Score(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 score, got %d", len(out))
	}
	if out[0].Model != "heuristic:v1" {
		t.Fatalf("expected heuristic model, got %s", out[0].Model)
	}
}

func TestScorerUsesLLMWhenAvailable(t *testing.T) {
	scorer := NewScorer(stubLLMScorer{scores: []SentimentScore{{
		ItemID:     1,
		Score:      0.8,
		Confidence: 0.9,
		Label:      "bullish",
		Reason:     "llm",
		Model:      "llm:gpt-4o-mini",
	}}}, 10)
	items := []domain.MarketIntelItem{{ID: 1, Title: "neutral", Excerpt: "neutral"}}

	out, err := scorer.Score(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out[0].Model != "llm:gpt-4o-mini" {
		t.Fatalf("expected llm model override, got %s", out[0].Model)
	}
	if out[0].Label != "bullish" {
		t.Fatalf("expected bullish label, got %s", out[0].Label)
	}
}

func TestScorerFallsBackWhenLLMErrors(t *testing.T) {
	scorer := NewScorer(stubLLMScorer{err: errors.New("boom")}, 10)
	items := []domain.MarketIntelItem{{ID: 1, Title: "hack and dump", Excerpt: "bear"}}

	out, err := scorer.Score(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out[0].Model != "heuristic:v1" {
		t.Fatalf("expected heuristic fallback, got %s", out[0].Model)
	}
}

type stubLLMScorer struct {
	scores []SentimentScore
	err    error
}

func (s stubLLMScorer) ScoreBatch(ctx context.Context, items []domain.MarketIntelItem) ([]SentimentScore, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]SentimentScore(nil), s.scores...), nil
}
