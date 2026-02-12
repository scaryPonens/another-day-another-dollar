package mcp

import (
	"fmt"
	"strings"

	"bug-free-umbrella/internal/domain"
)

const (
	defaultCandleLimit = 100
	maxCandleLimit     = 500
	defaultSignalLimit = 50
	maxSignalLimit     = 200
)

type pricesListLatestInput struct{}

type pricesListLatestOutput struct {
	Prices []*domain.PriceSnapshot `json:"prices"`
}

type pricesGetBySymbolInput struct {
	Symbol string `json:"symbol" jsonschema:"asset symbol (e.g. BTC, ETH)"`
}

type pricesGetBySymbolOutput struct {
	Price *domain.PriceSnapshot `json:"price"`
}

type candlesListInput struct {
	Symbol   string `json:"symbol" jsonschema:"asset symbol (e.g. BTC, ETH)"`
	Interval string `json:"interval" jsonschema:"candle interval: 5m, 15m, 1h, 4h, 1d"`
	Limit    int    `json:"limit,omitempty" jsonschema:"number of candles to return, max 500"`
}

type candlesListOutput struct {
	Symbol   string           `json:"symbol"`
	Interval string           `json:"interval"`
	Candles  []*domain.Candle `json:"candles"`
}

type signalsListInput struct {
	Symbol    string `json:"symbol,omitempty" jsonschema:"optional asset symbol (e.g. BTC, ETH)"`
	Risk      *int   `json:"risk,omitempty" jsonschema:"optional risk level 1-5"`
	Indicator string `json:"indicator,omitempty" jsonschema:"optional indicator: rsi, macd, bollinger, volume_zscore, ml_logreg_up4h, ml_xgboost_up4h, ml_ensemble_up4h"`
	Limit     int    `json:"limit,omitempty" jsonschema:"number of signals to return, max 200"`
}

type signalsListOutput struct {
	Signals []domain.Signal `json:"signals"`
}

type signalsGenerateInput struct {
	Symbol    string   `json:"symbol" jsonschema:"asset symbol (e.g. BTC, ETH)"`
	Intervals []string `json:"intervals,omitempty" jsonschema:"optional interval list: 5m,15m,1h,4h,1d"`
}

type signalsGenerateOutput struct {
	GeneratedCount int             `json:"generated_count"`
	Signals        []domain.Signal `json:"signals"`
}

func normalizeSymbol(symbol string) (string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return "", fmt.Errorf("symbol is required")
	}
	if _, ok := domain.CoinGeckoID[symbol]; !ok {
		return "", fmt.Errorf("unsupported symbol: %s", symbol)
	}
	return symbol, nil
}

func normalizeInterval(interval string) (string, error) {
	interval = strings.TrimSpace(interval)
	if interval == "" {
		return "", fmt.Errorf("interval is required")
	}
	for _, supported := range domain.SupportedIntervals {
		if interval == supported {
			return interval, nil
		}
	}
	return "", fmt.Errorf("unsupported interval: %s", interval)
}

func normalizeCandleLimit(limit int) int {
	if limit <= 0 {
		return defaultCandleLimit
	}
	if limit > maxCandleLimit {
		return maxCandleLimit
	}
	return limit
}

func normalizeSignalLimit(limit int) int {
	if limit <= 0 {
		return defaultSignalLimit
	}
	if limit > maxSignalLimit {
		return maxSignalLimit
	}
	return limit
}

func normalizeIndicator(indicator string) (string, error) {
	indicator = strings.ToLower(strings.TrimSpace(indicator))
	if indicator == "" {
		return "", nil
	}

	switch indicator {
	case domain.IndicatorRSI,
		domain.IndicatorMACD,
		domain.IndicatorBollinger,
		domain.IndicatorVolumeZ,
		domain.IndicatorMLLogRegUp4H,
		domain.IndicatorMLXGBoostUp4H,
		domain.IndicatorMLEnsembleUp4H:
		return indicator, nil
	default:
		return "", fmt.Errorf("unsupported indicator: %s", indicator)
	}
}

func normalizeSignalFilter(in signalsListInput) (domain.SignalFilter, error) {
	filter := domain.SignalFilter{Limit: normalizeSignalLimit(in.Limit)}

	if strings.TrimSpace(in.Symbol) != "" {
		symbol, err := normalizeSymbol(in.Symbol)
		if err != nil {
			return domain.SignalFilter{}, err
		}
		filter.Symbol = symbol
	}

	if in.Risk != nil {
		risk := domain.RiskLevel(*in.Risk)
		if !risk.IsValid() {
			return domain.SignalFilter{}, fmt.Errorf("risk must be between 1 and 5")
		}
		filter.Risk = &risk
	}

	indicator, err := normalizeIndicator(in.Indicator)
	if err != nil {
		return domain.SignalFilter{}, err
	}
	filter.Indicator = indicator

	return filter, nil
}

func normalizeGenerateIntervals(intervals []string) ([]string, error) {
	if len(intervals) == 0 {
		return append([]string(nil), domain.SupportedIntervals...), nil
	}

	seen := make(map[string]struct{}, len(intervals))
	result := make([]string, 0, len(intervals))
	for _, interval := range intervals {
		normalized, err := normalizeInterval(interval)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result, nil
}
