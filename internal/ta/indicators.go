package ta

import "math"

func MeanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func EMASeries(values []float64, period int) []float64 {
	if len(values) == 0 {
		return nil
	}
	if period <= 1 {
		out := make([]float64, len(values))
		copy(out, values)
		return out
	}
	out := make([]float64, len(values))
	alpha := 2.0 / float64(period+1)
	out[0] = values[0]
	for i := 1; i < len(values); i++ {
		out[i] = alpha*values[i] + (1-alpha)*out[i-1]
	}
	return out
}

func RSISeries(closes []float64, period int) []float64 {
	if len(closes) <= period {
		return nil
	}
	series := make([]float64, len(closes))
	for i := range series {
		series[i] = math.NaN()
	}

	var gainSum float64
	var lossSum float64
	for i := 1; i <= period; i++ {
		delta := closes[i] - closes[i-1]
		if delta > 0 {
			gainSum += delta
		} else {
			lossSum -= delta
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)
	series[period] = rsiFromAvg(avgGain, avgLoss)

	for i := period + 1; i < len(closes); i++ {
		delta := closes[i] - closes[i-1]
		gain := math.Max(delta, 0)
		loss := math.Max(-delta, 0)
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		series[i] = rsiFromAvg(avgGain, avgLoss)
	}
	return series
}

func rsiFromAvg(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func MACDSeries(values []float64, fast, slow, signal int) ([]float64, []float64) {
	if len(values) == 0 {
		return nil, nil
	}
	fastEMA := EMASeries(values, fast)
	slowEMA := EMASeries(values, slow)
	macdLine := make([]float64, len(values))
	for i := range values {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}
	signalLine := EMASeries(macdLine, signal)
	return macdLine, signalLine
}

func BollingerSeries(values []float64, period int, stdDevs float64) ([]float64, []float64, []float64) {
	if len(values) == 0 {
		return nil, nil, nil
	}
	middle := make([]float64, len(values))
	upper := make([]float64, len(values))
	lower := make([]float64, len(values))
	for i := range values {
		middle[i] = math.NaN()
		upper[i] = math.NaN()
		lower[i] = math.NaN()
	}
	if period <= 0 {
		return middle, upper, lower
	}
	for i := period - 1; i < len(values); i++ {
		window := values[i-period+1 : i+1]
		mean, std := MeanStd(window)
		middle[i] = mean
		upper[i] = mean + stdDevs*std
		lower[i] = mean - stdDevs*std
	}
	return middle, upper, lower
}
