package marketintel

import (
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestBuildCompositeDirectionAndRisk(t *testing.T) {
	out := BuildComposite(CompositeInput{
		Interval:       "1h",
		LongThreshold:  0.20,
		ShortThreshold: -0.20,
		FearGreed:      CompositeComponent{Score: 0.5, Confidence: 0.7, Available: true},
		News:           CompositeComponent{Score: 0.8, Confidence: 0.9, Available: true},
		Reddit:         CompositeComponent{Score: 0.2, Confidence: 0.6, Available: true},
		OnChain:        CompositeComponent{Score: 0.4, Confidence: 0.8, Available: true},
	})

	if out.Direction != domain.DirectionLong {
		t.Fatalf("expected long direction, got %s", out.Direction)
	}
	if out.Risk > domain.RiskLevel5 || out.Risk < domain.RiskLevel2 {
		t.Fatalf("unexpected risk mapping: %d", out.Risk)
	}
	if out.Weights["news"] == 0 || out.Weights["onchain"] == 0 {
		t.Fatalf("expected default weights to be present: %+v", out.Weights)
	}
}

func TestBuildCompositeRenormalizesMissingComponents(t *testing.T) {
	out := BuildComposite(CompositeInput{
		Interval:       "4h",
		LongThreshold:  0.20,
		ShortThreshold: -0.20,
		FearGreed:      CompositeComponent{Score: -0.3, Confidence: 0.7, Available: true},
		News:           CompositeComponent{Score: -0.6, Confidence: 0.8, Available: true},
	})

	if len(out.Weights) != 2 {
		t.Fatalf("expected only two active weights, got %+v", out.Weights)
	}
	if out.Direction != domain.DirectionShort {
		t.Fatalf("expected short direction from negative score, got %s", out.Direction)
	}
	if out.Weights["fear_greed"]+out.Weights["news"] < 0.99 {
		t.Fatalf("expected normalized weights to sum near 1, got %+v", out.Weights)
	}
}

func TestBuildCompositeNoComponents(t *testing.T) {
	out := BuildComposite(CompositeInput{Interval: "1h", LongThreshold: 0.2, ShortThreshold: -0.2})
	if out.Direction != domain.DirectionHold {
		t.Fatalf("expected hold with empty inputs, got %s", out.Direction)
	}
	if out.Risk != domain.RiskLevel5 {
		t.Fatalf("expected risk 5 for empty inputs, got %d", out.Risk)
	}
}
