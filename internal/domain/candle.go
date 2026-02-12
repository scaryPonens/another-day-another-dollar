package domain

import "time"

// Candle represents a single OHLCV candle for an asset at a given interval.
type Candle struct {
	Symbol   string    `json:"symbol"`
	Interval string    `json:"interval"`
	OpenTime time.Time `json:"open_time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume"`
}

// PriceSnapshot represents the latest price data for an asset.
type PriceSnapshot struct {
	Symbol          string  `json:"symbol"`
	PriceUSD        float64 `json:"price_usd"`
	Volume24h       float64 `json:"volume_24h"`
	Change24hPct    float64 `json:"change_24h_pct"`
	LastUpdatedUnix int64   `json:"last_updated_unix"`
}

// CoinGeckoID maps internal symbols to CoinGecko API identifiers.
var CoinGeckoID = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"SOL":   "solana",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOGE":  "dogecoin",
	"DOT":   "polkadot",
	"AVAX":  "avalanche-2",
	"LINK":  "chainlink",
	"MATIC": "matic-network",
}

// CoinGeckoIDToSymbol is the reverse mapping.
var CoinGeckoIDToSymbol map[string]string

func init() {
	CoinGeckoIDToSymbol = make(map[string]string, len(CoinGeckoID))
	for sym, id := range CoinGeckoID {
		CoinGeckoIDToSymbol[id] = sym
	}
}

// SupportedSymbols lists all tracked crypto symbols.
var SupportedSymbols = []string{
	"BTC", "ETH", "SOL", "XRP", "ADA",
	"DOGE", "DOT", "AVAX", "LINK", "MATIC",
}

// SupportedIntervals defines the candle intervals we store.
var SupportedIntervals = []string{"5m", "15m", "1h", "4h", "1d"}
