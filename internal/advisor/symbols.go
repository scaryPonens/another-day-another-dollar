package advisor

import (
	"strings"

	"bug-free-umbrella/internal/domain"
)

// ExtractSymbols scans the user message for mentions of supported crypto symbols.
// Returns deduplicated uppercase symbols found.
func ExtractSymbols(text string) []string {
	upper := strings.ToUpper(text)
	words := strings.FieldsFunc(upper, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	seen := make(map[string]bool)
	var result []string
	for _, w := range words {
		if _, ok := domain.CoinGeckoID[w]; ok && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}
