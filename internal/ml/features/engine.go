package features

import (
	"math"
	"sort"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/ta"
)

const (
	featureSpecVersion = "v1"
	rsiPeriod          = 14
	macdFast           = 12
	macdSlow           = 26
	macdSignal         = 9
	bbPeriod           = 20
	bbStdDevs          = 2.0
)

type Engine struct {
	now func() time.Time
}

func NewEngine(now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{now: now}
}

func FeatureSpecVersion() string {
	return featureSpecVersion
}

func (e *Engine) BuildRows(candles []*domain.Candle, targetHours int) []domain.MLFeatureRow {
	normalized := normalizeCandles(candles)
	if len(normalized) == 0 {
		return nil
	}
	if targetHours <= 0 {
		targetHours = 4
	}

	closes := make([]float64, len(normalized))
	volumes := make([]float64, len(normalized))
	for i := range normalized {
		closes[i] = normalized[i].Close
		volumes[i] = normalized[i].Volume
	}

	rsi := ta.RSISeries(closes, rsiPeriod)
	macdLine, macdSig := ta.MACDSeries(closes, macdFast, macdSlow, macdSignal)
	bbMiddle, bbUpper, bbLower := ta.BollingerSeries(closes, bbPeriod, bbStdDevs)

	now := e.now().UTC()
	rows := make([]domain.MLFeatureRow, 0, len(normalized))
	for i := range normalized {
		if i < 24 || i >= len(normalized)-1 {
			continue
		}

		ret1h := pctReturn(closes, i, 1)
		ret4h := pctReturn(closes, i, 4)
		ret12h := pctReturn(closes, i, 12)
		ret24h := pctReturn(closes, i, 24)
		if anyNaN(ret1h, ret4h, ret12h, ret24h) {
			continue
		}

		vol6h := rollingVolatility(closes, i, 6)
		vol24h := rollingVolatility(closes, i, 24)
		if anyNaN(vol6h, vol24h) {
			continue
		}

		volZ24 := rollingZ(volumes, i, 24)
		if math.IsNaN(volZ24) {
			continue
		}

		if i >= len(rsi) || i >= len(macdLine) || i >= len(macdSig) || i >= len(bbUpper) || i >= len(bbLower) || i >= len(bbMiddle) {
			continue
		}
		rsiVal := rsi[i]
		macdL := macdLine[i]
		macdS := macdSig[i]
		bbU := bbUpper[i]
		bbL := bbLower[i]
		bbM := bbMiddle[i]
		if anyNaN(rsiVal, macdL, macdS, bbU, bbL, bbM) {
			continue
		}
		bbWidth := 0.0
		if bbM != 0 {
			bbWidth = (bbU - bbL) / bbM
		}
		bbPos := 0.5
		if bbU != bbL {
			bbPos = (closes[i] - bbL) / (bbU - bbL)
		}

		var target *bool
		targetIdx := i + targetHours
		if targetIdx < len(closes) {
			up := closes[targetIdx] > closes[i]
			target = &up
		}

		rows = append(rows, domain.MLFeatureRow{
			Symbol:        normalized[i].Symbol,
			Interval:      normalized[i].Interval,
			OpenTime:      normalized[i].OpenTime.UTC(),
			Ret1H:         ret1h,
			Ret4H:         ret4h,
			Ret12H:        ret12h,
			Ret24H:        ret24h,
			Volatility6H:  vol6h,
			Volatility24H: vol24h,
			VolumeZ24H:    volZ24,
			RSI14:         rsiVal,
			MACDLine:      macdL,
			MACDSignal:    macdS,
			MACDHist:      macdL - macdS,
			BBPos:         bbPos,
			BBWidth:       bbWidth,
			TargetUp4H:    target,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return rows
}

func normalizeCandles(in []*domain.Candle) []domain.Candle {
	out := make([]domain.Candle, 0, len(in))
	for _, c := range in {
		if c == nil {
			continue
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].OpenTime.Before(out[j].OpenTime)
	})
	return out
}

func pctReturn(values []float64, idx int, lag int) float64 {
	if idx-lag < 0 || idx >= len(values) {
		return math.NaN()
	}
	base := values[idx-lag]
	if base == 0 {
		return math.NaN()
	}
	return (values[idx] / base) - 1
}

func rollingVolatility(closes []float64, idx int, window int) float64 {
	if window <= 1 || idx-window+1 <= 0 || idx >= len(closes) {
		return math.NaN()
	}
	rets := make([]float64, 0, window)
	for j := idx - window + 1; j <= idx; j++ {
		if j-1 < 0 || closes[j-1] == 0 {
			return math.NaN()
		}
		rets = append(rets, (closes[j]/closes[j-1])-1)
	}
	_, std := ta.MeanStd(rets)
	return std
}

func rollingZ(values []float64, idx int, window int) float64 {
	if window <= 0 || idx-window < 0 || idx >= len(values) {
		return math.NaN()
	}
	mean, std := ta.MeanStd(values[idx-window : idx])
	if std == 0 {
		return 0
	}
	return (values[idx] - mean) / std
}

func anyNaN(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return true
		}
	}
	return false
}
