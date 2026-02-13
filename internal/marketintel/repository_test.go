package marketintel

import (
	"reflect"
	"testing"
)

func TestNormalizeSymbolList(t *testing.T) {
	got := normalizeSymbolList([]string{"btc", "ETH", "ETH", "fake", ""})
	expected := []string{"BTC", "ETH"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestEnsureJSON(t *testing.T) {
	if ensureJSON("") != "{}" {
		t.Fatalf("empty json should default to {}")
	}
	if ensureJSON("{\"ok\":true}") != "{\"ok\":true}" {
		t.Fatalf("valid json should stay unchanged")
	}
	got := ensureJSON("not-json")
	if got == "not-json" || got == "{}" {
		t.Fatalf("invalid json should be wrapped, got %s", got)
	}
}
