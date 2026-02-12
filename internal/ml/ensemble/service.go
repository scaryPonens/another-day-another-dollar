package ensemble

import "bug-free-umbrella/internal/domain"

type Components struct {
	ClassicScore float64
	LogRegProb   float64
	XGBoostProb  float64
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Score(c Components) float64 {
	logRegScore := 2*c.LogRegProb - 1
	xgbScore := 2*c.XGBoostProb - 1
	return 0.30*c.ClassicScore + 0.35*logRegScore + 0.35*xgbScore
}

func Direction(score float64) domain.SignalDirection {
	if score > 0.15 {
		return domain.DirectionLong
	}
	if score < -0.15 {
		return domain.DirectionShort
	}
	return domain.DirectionHold
}
