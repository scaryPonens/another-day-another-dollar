package advisor

import (
	"testing"
)

func TestExtractSymbolsSingleMention(t *testing.T) {
	got := ExtractSymbols("What about SOL?")
	if len(got) != 1 || got[0] != "SOL" {
		t.Fatalf("expected [SOL], got %v", got)
	}
}

func TestExtractSymbolsMultipleMentions(t *testing.T) {
	got := ExtractSymbols("Compare BTC and ETH")
	if len(got) != 2 {
		t.Fatalf("expected 2 symbols, got %v", got)
	}
	symbols := map[string]bool{}
	for _, s := range got {
		symbols[s] = true
	}
	if !symbols["BTC"] || !symbols["ETH"] {
		t.Fatalf("expected BTC and ETH, got %v", got)
	}
}

func TestExtractSymbolsNoMention(t *testing.T) {
	got := ExtractSymbols("What looks good right now?")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestExtractSymbolsCaseInsensitive(t *testing.T) {
	got := ExtractSymbols("how's sol doing?")
	if len(got) != 1 || got[0] != "SOL" {
		t.Fatalf("expected [SOL], got %v", got)
	}
}

func TestExtractSymbolsDeduplication(t *testing.T) {
	got := ExtractSymbols("BTC BTC BTC is the best BTC")
	if len(got) != 1 || got[0] != "BTC" {
		t.Fatalf("expected [BTC], got %v", got)
	}
}

func TestExtractSymbolsInSentence(t *testing.T) {
	got := ExtractSymbols("Should I buy DOGE or stick with LINK?")
	if len(got) != 2 {
		t.Fatalf("expected 2 symbols, got %v", got)
	}
	symbols := map[string]bool{}
	for _, s := range got {
		symbols[s] = true
	}
	if !symbols["DOGE"] || !symbols["LINK"] {
		t.Fatalf("expected DOGE and LINK, got %v", got)
	}
}
