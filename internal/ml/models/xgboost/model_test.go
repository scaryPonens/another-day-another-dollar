package xgboost

import (
	"testing"
)

func TestTrainPredictAndRoundTrip(t *testing.T) {
	samples, labels := dataset()
	model, err := Train(samples, labels, []string{"x1", "x2"}, DefaultTrainOptions())
	if err != nil {
		t.Fatalf("train failed: %v", err)
	}

	pLow := model.PredictProb([]float64{-1.8, -1.3})
	pHigh := model.PredictProb([]float64{1.8, 1.3})
	if pLow < 0 || pLow > 1 || pHigh < 0 || pHigh > 1 {
		t.Fatalf("expected probabilities in [0,1], got low=%.4f high=%.4f", pLow, pHigh)
	}
	if pHigh <= pLow {
		t.Fatalf("expected positive sample probability > negative sample probability, got %.4f <= %.4f", pHigh, pLow)
	}

	blob, err := model.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	restored, err := UnmarshalBinary(blob)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	pRoundTrip := restored.PredictProb([]float64{1.8, 1.3})
	if pRoundTrip < 0 || pRoundTrip > 1 {
		t.Fatalf("expected roundtrip probability in [0,1], got %.4f", pRoundTrip)
	}
}

func dataset() ([][]float64, []float64) {
	samples := make([][]float64, 0, 120)
	labels := make([]float64, 0, 120)
	for i := 0; i < 60; i++ {
		samples = append(samples, []float64{-2.0 + float64(i)/90.0, -1.5 + float64(i)/120.0})
		labels = append(labels, 0)
	}
	for i := 0; i < 60; i++ {
		samples = append(samples, []float64{1.0 + float64(i)/90.0, 1.1 + float64(i)/110.0})
		labels = append(labels, 1)
	}
	return samples, labels
}
