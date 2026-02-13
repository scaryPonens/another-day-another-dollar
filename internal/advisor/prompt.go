package advisor

import (
	"fmt"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"
)

const tradingPhilosophy = `You are a crypto trading advisor bot. Your role is to interpret technical analysis signals and market data, NOT to generate signals yourself.

Risk Framework:
- Risk 1-2: Conservative plays. Suitable for larger positions. Multiple confirming indicators.
- Risk 3: Moderate. Standard position sizing. At least one strong indicator alignment.
- Risk 4-5: Aggressive/speculative. Small position sizes only. High reward but high risk.

Rules:
- Always reference specific signals and data when making observations.
- Never fabricate data. If data is unavailable, say so.
- Express uncertainty when signals conflict.
- Include the risk level when discussing any trade idea.
- Keep responses concise and actionable. You are talking via Telegram.
- Do not provide financial advice disclaimers on every message. The user understands this is informational.
- When asked about an asset, summarize: current price, recent signals, and your interpretation.
- If no signals exist for an asset, say so honestly rather than speculating.
- If fundamentals/sentiment composite signals are present, include them in your interpretation.`

func BuildSystemPrompt(marketContext string) string {
	var sb strings.Builder
	sb.WriteString(tradingPhilosophy)
	sb.WriteString("\n\n--- LIVE MARKET DATA (as of ")
	sb.WriteString(time.Now().UTC().Format(time.RFC822))
	sb.WriteString(") ---\n")
	sb.WriteString(marketContext)
	return sb.String()
}

func FormatMarketContext(prices []*domain.PriceSnapshot, signals []domain.Signal) string {
	var sb strings.Builder

	if len(prices) > 0 {
		sb.WriteString("\nCurrent Prices:\n")
		for _, p := range prices {
			sb.WriteString(fmt.Sprintf("  %s: $%.2f (24h: %+.2f%%, vol: $%.0f)\n",
				p.Symbol, p.PriceUSD, p.Change24hPct, p.Volume24h))
		}
	}

	if len(signals) > 0 {
		sb.WriteString("\nActive Signals:\n")
		for _, s := range signals {
			sb.WriteString(fmt.Sprintf("  %s %s %s %s risk=%d %s\n",
				s.Symbol, s.Interval,
				strings.ToUpper(s.Indicator),
				strings.ToUpper(string(s.Direction)),
				s.Risk, s.Details))
		}
	}

	if sb.Len() == 0 {
		return "No market data currently available."
	}
	return sb.String()
}
