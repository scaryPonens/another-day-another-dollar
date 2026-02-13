package marketintel

import (
	"regexp"
	"sort"
	"strings"

	"bug-free-umbrella/internal/domain"
)

var symbolTokenRx = regexp.MustCompile(`\$?[A-Za-z]{2,10}`)

var symbolAlias = map[string][]string{
	"BTC":   {"btc", "bitcoin", "xbt"},
	"ETH":   {"eth", "ethereum"},
	"SOL":   {"sol", "solana"},
	"XRP":   {"xrp", "ripple", "xrpl"},
	"ADA":   {"ada", "cardano"},
	"DOGE":  {"doge", "dogecoin"},
	"DOT":   {"dot", "polkadot"},
	"AVAX":  {"avax", "avalanche"},
	"LINK":  {"link", "chainlink"},
	"MATIC": {"matic", "polygon"},
}

var subredditSymbolHint = map[string]string{
	"bitcoin":        "BTC",
	"ethereum":       "ETH",
	"cardano":        "ADA",
	"ripple":         "XRP",
	"xrpl":           "XRP",
	"cryptocurrency": "",
}

func ExtractSymbolsFromContent(source, title, excerpt string, metadata map[string]any) []string {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "fear_greed" {
		return append([]string(nil), domain.SupportedSymbols...)
	}

	text := strings.ToLower(strings.Join([]string{title, excerpt}, " "))
	matched := make(map[string]struct{}, 8)

	for _, raw := range symbolTokenRx.FindAllString(text, -1) {
		token := strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(raw), "$"))
		if _, ok := domain.CoinGeckoID[token]; ok {
			matched[token] = struct{}{}
		}
	}

	for symbol, aliases := range symbolAlias {
		for _, alias := range aliases {
			if strings.Contains(text, alias) {
				matched[symbol] = struct{}{}
				break
			}
		}
	}

	if metadata != nil {
		if subreddit, ok := metadata["subreddit"].(string); ok {
			if symbol, found := subredditSymbolHint[strings.ToLower(strings.TrimSpace(subreddit))]; found && symbol != "" {
				matched[symbol] = struct{}{}
			}
		}
	}

	if len(matched) == 0 {
		return nil
	}
	out := make([]string, 0, len(matched))
	for symbol := range matched {
		out = append(out, symbol)
	}
	sort.Strings(out)
	return out
}
