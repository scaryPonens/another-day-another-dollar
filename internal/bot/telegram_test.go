package bot

import (
	"testing"

	"bug-free-umbrella/internal/domain"
)

func TestStartTelegramBotSkipsWithoutToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	StartTelegramBot(nil, nil, nil)
}

func TestParseSignalArgsSymbolAndRisk(t *testing.T) {
	filter, err := parseSignalArgs([]string{"btc", "--risk", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filter.Symbol != "BTC" {
		t.Fatalf("expected symbol BTC, got %s", filter.Symbol)
	}
	if filter.Risk == nil || *filter.Risk != domain.RiskLevel3 {
		t.Fatalf("expected risk level 3, got %+v", filter.Risk)
	}
	if filter.Limit != 5 {
		t.Fatalf("expected default limit=5, got %d", filter.Limit)
	}
}

func TestParseSignalArgsRejectsInvalidRisk(t *testing.T) {
	if _, err := parseSignalArgs([]string{"--risk", "8"}); err == nil {
		t.Fatal("expected risk parsing error")
	}
}
