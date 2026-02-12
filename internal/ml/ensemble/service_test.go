package ensemble

import (
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestScoreAndDirection(t *testing.T) {
	s := NewService()
	score := s.Score(Components{
		ClassicScore: 0.5,
		LogRegProb:   0.7,
		XGBoostProb:  0.8,
	})
	if score <= 0.15 {
		t.Fatalf("expected bullish score > 0.15, got %.4f", score)
	}
	if dir := Direction(score); dir != domain.DirectionLong {
		t.Fatalf("expected long direction, got %s", dir)
	}

	score = s.Score(Components{
		ClassicScore: -0.7,
		LogRegProb:   0.3,
		XGBoostProb:  0.2,
	})
	if score >= -0.15 {
		t.Fatalf("expected bearish score < -0.15, got %.4f", score)
	}
	if dir := Direction(score); dir != domain.DirectionShort {
		t.Fatalf("expected short direction, got %s", dir)
	}
}
