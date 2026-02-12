package common

import (
	"math"

	"bug-free-umbrella/internal/domain"
)

const (
	ModelKeyLogReg     = "logreg"
	ModelKeyXGBoost    = "xgboost"
	ModelKeyEnsembleV1 = "ensemble_v1"
)

var FeatureNames = []string{
	"ret_1h",
	"ret_4h",
	"ret_12h",
	"ret_24h",
	"volatility_6h",
	"volatility_24h",
	"volume_z_24h",
	"rsi_14",
	"macd_line",
	"macd_signal",
	"macd_hist",
	"bb_pos",
	"bb_width",
}

func FeatureVector(row domain.MLFeatureRow) []float64 {
	return []float64{
		row.Ret1H,
		row.Ret4H,
		row.Ret12H,
		row.Ret24H,
		row.Volatility6H,
		row.Volatility24H,
		row.VolumeZ24H,
		row.RSI14,
		row.MACDLine,
		row.MACDSignal,
		row.MACDHist,
		row.BBPos,
		row.BBWidth,
	}
}

func TargetLabel(row domain.MLFeatureRow) (float64, bool) {
	if row.TargetUp4H == nil {
		return 0, false
	}
	if *row.TargetUp4H {
		return 1, true
	}
	return 0, true
}

func Clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0.5
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func Confidence(probUp float64) float64 {
	return math.Abs(Clamp01(probUp)-0.5) * 2
}

func RiskFromConfidence(confidence float64) domain.RiskLevel {
	switch {
	case confidence >= 0.80:
		return domain.RiskLevel2
	case confidence >= 0.60:
		return domain.RiskLevel3
	case confidence >= 0.40:
		return domain.RiskLevel4
	default:
		return domain.RiskLevel5
	}
}

func DirectionFromProb(probUp, longThreshold, shortThreshold float64) domain.SignalDirection {
	probUp = Clamp01(probUp)
	if probUp >= longThreshold {
		return domain.DirectionLong
	}
	if probUp <= shortThreshold {
		return domain.DirectionShort
	}
	return domain.DirectionHold
}
