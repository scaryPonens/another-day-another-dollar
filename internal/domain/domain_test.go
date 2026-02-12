package domain

import (
	"testing"
	"time"
)

func TestRiskLevelConstants(t *testing.T) {
	if RiskLevel1 != 1 || RiskLevel5 != 5 {
		t.Errorf("RiskLevel constants not set correctly: got %d, %d", RiskLevel1, RiskLevel5)
	}
}

func TestAssetFields(t *testing.T) {
	a := Asset{Symbol: "BTC", Name: "Bitcoin"}
	if a.Symbol != "BTC" || a.Name != "Bitcoin" {
		t.Errorf("Asset fields not set correctly: %+v", a)
	}
}

func TestSignalFields(t *testing.T) {
	ts := time.Unix(1234567890, 0).UTC()
	s := Signal{
		Symbol:    "ETH",
		Interval:  "4h",
		Indicator: IndicatorRSI,
		Timestamp: ts,
		Risk:      RiskLevel3,
		Direction: DirectionLong,
	}
	if s.Symbol != "ETH" || s.Interval != "4h" || s.Indicator != IndicatorRSI || !s.Timestamp.Equal(ts) || s.Risk != RiskLevel3 || s.Direction != DirectionLong {
		t.Errorf("Signal fields not set correctly: %+v", s)
	}
}

func TestRecommendationFields(t *testing.T) {
	s := Signal{Symbol: "SOL", Indicator: IndicatorMACD}
	r := Recommendation{Signal: s, Text: "Buy"}
	if r.Signal.Indicator != IndicatorMACD || r.Text != "Buy" {
		t.Errorf("Recommendation fields not set correctly: %+v", r)
	}
}

func TestRiskLevelIsValid(t *testing.T) {
	if !RiskLevel1.IsValid() || !RiskLevel5.IsValid() {
		t.Fatal("expected boundary values to be valid")
	}
	if RiskLevel(0).IsValid() || RiskLevel(6).IsValid() {
		t.Fatal("expected out-of-range risk levels to be invalid")
	}
}

func TestMLIndicatorConstants(t *testing.T) {
	if IndicatorMLLogRegUp4H == "" || IndicatorMLXGBoostUp4H == "" || IndicatorMLEnsembleUp4H == "" {
		t.Fatal("expected ML indicator constants to be non-empty")
	}
}
