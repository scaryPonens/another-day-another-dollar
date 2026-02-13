package marketintel

import (
	"reflect"
	"testing"
)

func TestExtractSymbolsFromContentKeywords(t *testing.T) {
	symbols := ExtractSymbolsFromContent("news", "Bitcoin and ETH rally", "$ADA joins move", nil)
	expected := []string{"ADA", "BTC", "ETH"}
	if !reflect.DeepEqual(symbols, expected) {
		t.Fatalf("expected %v, got %v", expected, symbols)
	}
}

func TestExtractSymbolsFromContentSubredditHint(t *testing.T) {
	symbols := ExtractSymbolsFromContent("reddit", "Daily thread", "no explicit token", map[string]any{"subreddit": "Ripple"})
	expected := []string{"XRP"}
	if !reflect.DeepEqual(symbols, expected) {
		t.Fatalf("expected %v, got %v", expected, symbols)
	}
}

func TestExtractSymbolsFromFearGreed(t *testing.T) {
	symbols := ExtractSymbolsFromContent("fear_greed", "", "", nil)
	if len(symbols) != 10 {
		t.Fatalf("expected all supported symbols, got %d", len(symbols))
	}
}
