package marketintel

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"bug-free-umbrella/internal/domain"
)

const modelKeyFundSentV1 = "fund_sent_v1"

type CompositeComponent struct {
	Score      float64
	Confidence float64
	Available  bool
}

type CompositeInput struct {
	Interval       string
	LongThreshold  float64
	ShortThreshold float64

	FearGreedValue *int
	FearGreed      CompositeComponent
	News           CompositeComponent
	Reddit         CompositeComponent
	OnChain        CompositeComponent
}

type CompositeResult struct {
	Score       float64
	Confidence  float64
	Direction   domain.SignalDirection
	Risk        domain.RiskLevel
	Weights     map[string]float64
	DetailsText string
}

func BuildComposite(in CompositeInput) CompositeResult {
	weights := map[string]float64{
		"fear_greed": 0.20,
		"news":       0.35,
		"reddit":     0.25,
		"onchain":    0.20,
	}

	components := map[string]CompositeComponent{
		"fear_greed": in.FearGreed,
		"news":       in.News,
		"reddit":     in.Reddit,
		"onchain":    in.OnChain,
	}

	activeWeight := 0.0
	for name, c := range components {
		if c.Available {
			activeWeight += weights[name]
		}
	}

	if activeWeight <= 0 {
		return CompositeResult{
			Score:       0,
			Confidence:  0,
			Direction:   domain.DirectionHold,
			Risk:        domain.RiskLevel5,
			Weights:     map[string]float64{},
			DetailsText: fmt.Sprintf("model_key=%s;interval=%s;score=0.0000;confidence=0.0000;fng=na;news=na;reddit=na;onchain=na", modelKeyFundSentV1, in.Interval),
		}
	}

	normalized := make(map[string]float64, len(weights))
	for name := range weights {
		if !components[name].Available {
			continue
		}
		normalized[name] = weights[name] / activeWeight
	}

	score := 0.0
	confidence := 0.0
	for name, w := range normalized {
		score += w * clamp(components[name].Score, -1, 1)
		confidence += w * clamp(components[name].Confidence, 0, 1)
	}
	score = clamp(score, -1, 1)
	confidence = clamp(confidence, 0, 1)

	direction := domain.DirectionHold
	if score >= in.LongThreshold {
		direction = domain.DirectionLong
	} else if score <= in.ShortThreshold {
		direction = domain.DirectionShort
	}

	conviction := math.Abs(score) * confidence
	risk := domain.RiskLevel5
	switch {
	case conviction >= 0.70:
		risk = domain.RiskLevel2
	case conviction >= 0.50:
		risk = domain.RiskLevel3
	case conviction >= 0.30:
		risk = domain.RiskLevel4
	default:
		risk = domain.RiskLevel5
	}

	details := formatDetails(in, score, confidence)
	return CompositeResult{
		Score:       score,
		Confidence:  confidence,
		Direction:   direction,
		Risk:        risk,
		Weights:     normalized,
		DetailsText: details,
	}
}

func formatDetails(in CompositeInput, score, confidence float64) string {
	fng := "na"
	if in.FearGreed.Available {
		fng = fmt.Sprintf("%.4f", clamp(in.FearGreed.Score, -1, 1))
	}
	news := "na"
	if in.News.Available {
		news = fmt.Sprintf("%.4f", clamp(in.News.Score, -1, 1))
	}
	reddit := "na"
	if in.Reddit.Available {
		reddit = fmt.Sprintf("%.4f", clamp(in.Reddit.Score, -1, 1))
	}
	onchain := "na"
	if in.OnChain.Available {
		onchain = fmt.Sprintf("%.4f", clamp(in.OnChain.Score, -1, 1))
	}

	parts := []string{
		fmt.Sprintf("model_key=%s", modelKeyFundSentV1),
		fmt.Sprintf("interval=%s", in.Interval),
		fmt.Sprintf("score=%.4f", score),
		fmt.Sprintf("confidence=%.4f", confidence),
		fmt.Sprintf("fng=%s", fng),
		fmt.Sprintf("news=%s", news),
		fmt.Sprintf("reddit=%s", reddit),
		fmt.Sprintf("onchain=%s", onchain),
	}
	if in.FearGreedValue != nil {
		parts = append(parts, fmt.Sprintf("fng_value=%d", *in.FearGreedValue))
	}
	sort.Strings(parts[4:])
	return strings.Join(parts, ";")
}

func clamp(v, lo, hi float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
