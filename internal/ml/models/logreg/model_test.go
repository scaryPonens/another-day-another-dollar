package logreg

import (
	"math"
	"testing"
)

func TestTrainPredictAndRoundTrip(t *testing.T) {
	samples, labels := separableData()
	model, err := Train(samples, labels, []string{"x1", "x2"}, DefaultTrainOptions())
	if err != nil {
		t.Fatalf("train failed: %v", err)
	}

	pLow := model.PredictProb([]float64{-2, -2})
	pHigh := model.PredictProb([]float64{3, 3})
	if pLow >= 0.5 {
		t.Fatalf("expected low sample prob < 0.5, got %.4f", pLow)
	}
	if pHigh <= 0.5 {
		t.Fatalf("expected high sample prob > 0.5, got %.4f", pHigh)
	}

	blob, err := model.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	restored, err := UnmarshalBinary(blob)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if diff := math.Abs(restored.PredictProb([]float64{3, 3}) - pHigh); diff > 1e-6 {
		t.Fatalf("roundtrip changed prediction by %.8f", diff)
	}
}

func separableData() ([][]float64, []float64) {
	samples := make([][]float64, 0, 80)
	labels := make([]float64, 0, 80)
	for i := 0; i < 40; i++ {
		samples = append(samples, []float64{-1.5 - float64(i)/40, -1.0 - float64(i)/60})
		labels = append(labels, 0)
	}
	for i := 0; i < 40; i++ {
		samples = append(samples, []float64{1.0 + float64(i)/40, 1.4 + float64(i)/60})
		labels = append(labels, 1)
	}
	return samples, labels
}
