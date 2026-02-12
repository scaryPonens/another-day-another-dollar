package domain

import (
	"testing"
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
	a := Asset{Symbol: "ETH", Name: "Ethereum"}
	s := Signal{
		Asset:     a,
		Indicator: "RSI",
		Timestamp: 1234567890,
		Risk:      RiskLevel3,
		Direction: "long",
	}
	if s.Asset.Symbol != "ETH" || s.Indicator != "RSI" || s.Risk != RiskLevel3 || s.Direction != "long" {
		t.Errorf("Signal fields not set correctly: %+v", s)
	}
}

func TestRecommendationFields(t *testing.T) {
	s := Signal{Asset: Asset{Symbol: "SOL"}, Indicator: "MACD"}
	r := Recommendation{Signal: s, Text: "Buy"}
	if r.Signal.Indicator != "MACD" || r.Text != "Buy" {
		t.Errorf("Recommendation fields not set correctly: %+v", r)
	}
}
