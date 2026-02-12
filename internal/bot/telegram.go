package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"

	tele "gopkg.in/telebot.v3"
)

type PriceQuerier interface {
	GetCurrentPrice(ctx context.Context, symbol string) (*domain.PriceSnapshot, error)
}

type SignalLister interface {
	ListSignals(ctx context.Context, filter domain.SignalFilter) ([]domain.Signal, error)
}

type Advisor interface {
	Ask(ctx context.Context, chatID int64, message string) (string, error)
}

func StartTelegramBot(priceService PriceQuerier, signalService SignalLister, advisorService Advisor) *AlertDispatcher {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Println("TELEGRAM_BOT_TOKEN not set, skipping Telegram bot startup")
		return nil
	}
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %v", err)
	}
	alerts := NewAlertDispatcher(b)

	b.Handle("/ping", func(c tele.Context) error {
		return c.Send("pong")
	})

	b.Handle("/price", func(c tele.Context) error {
		args := c.Args()
		if len(args) == 0 {
			return c.Send(fmt.Sprintf("Usage: /price BTC\nSupported: %s", strings.Join(domain.SupportedSymbols, ", ")))
		}
		symbol := strings.ToUpper(args[0])
		if _, ok := domain.CoinGeckoID[symbol]; !ok {
			return c.Send(fmt.Sprintf("Unknown symbol: %s\nSupported: %s", symbol, strings.Join(domain.SupportedSymbols, ", ")))
		}
		snapshot, err := priceService.GetCurrentPrice(context.Background(), symbol)
		if err != nil {
			return c.Send(fmt.Sprintf("Error fetching price for %s: %v", symbol, err))
		}
		msg := fmt.Sprintf(
			"%s\nPrice: $%.2f\n24h Change: %.2f%%\n24h Volume: $%.0f",
			symbol, snapshot.PriceUSD, snapshot.Change24hPct, snapshot.Volume24h,
		)
		return c.Send(msg)
	})

	b.Handle("/volume", func(c tele.Context) error {
		args := c.Args()
		if len(args) == 0 {
			return c.Send(fmt.Sprintf("Usage: /volume SOL\nSupported: %s", strings.Join(domain.SupportedSymbols, ", ")))
		}
		symbol := strings.ToUpper(args[0])
		if _, ok := domain.CoinGeckoID[symbol]; !ok {
			return c.Send(fmt.Sprintf("Unknown symbol: %s\nSupported: %s", symbol, strings.Join(domain.SupportedSymbols, ", ")))
		}
		snapshot, err := priceService.GetCurrentPrice(context.Background(), symbol)
		if err != nil {
			return c.Send(fmt.Sprintf("Error fetching volume for %s: %v", symbol, err))
		}
		msg := fmt.Sprintf(
			"%s 24h Trading Volume\nVolume: $%.0f\nPrice: $%.2f\n24h Change: %.2f%%",
			symbol, snapshot.Volume24h, snapshot.PriceUSD, snapshot.Change24hPct,
		)
		return c.Send(msg)
	})

	b.Handle("/signals", func(c tele.Context) error {
		if signalService == nil {
			return c.Send("Signal service unavailable")
		}

		filter, err := parseSignalArgs(c.Args())
		if err != nil {
			return c.Send("Usage: /signals BTC | /signals --risk 3 | /signals BTC --risk 3")
		}

		signals, err := signalService.ListSignals(context.Background(), filter)
		if err != nil {
			return c.Send(fmt.Sprintf("Error fetching signals: %v", err))
		}
		if len(signals) == 0 {
			return c.Send("No matching signals right now.")
		}

		lines := make([]string, 0, len(signals)+1)
		lines = append(lines, "Latest signals:")
		for _, s := range signals {
			lines = append(lines, formatSignal(s))
		}
		return c.Send(strings.Join(lines, "\n"))
	})

	b.Handle("/alerts", func(c tele.Context) error {
		chat := c.Chat()
		if chat == nil {
			return c.Send("Unable to detect chat")
		}

		mode, err := parseAlertMode(c.Args())
		if err != nil {
			return c.Send("Usage: /alerts on | /alerts off | /alerts status")
		}

		switch mode {
		case "on":
			if alerts.Subscribe(chat.ID) {
				return c.Send("Proactive alerts enabled for this chat.")
			}
			return c.Send("Proactive alerts are already enabled for this chat.")
		case "off":
			if alerts.Unsubscribe(chat.ID) {
				return c.Send("Proactive alerts disabled for this chat.")
			}
			return c.Send("Proactive alerts are already disabled for this chat.")
		default:
			if alerts.IsSubscribed(chat.ID) {
				return c.Send("Alerts status: ON")
			}
			return c.Send("Alerts status: OFF")
		}
	})

	b.Handle("/ask", func(c tele.Context) error {
		if advisorService == nil {
			return c.Send("Advisor not configured. Set OPENAI_API_KEY to enable.")
		}
		question := strings.TrimSpace(c.Message().Payload)
		if question == "" {
			return c.Send("Usage: /ask <question>\nExample: /ask What do you think about BTC?")
		}
		return handleAdvisorQuery(c, advisorService, question)
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		if advisorService == nil {
			return nil
		}
		text := strings.TrimSpace(c.Text())
		if text == "" {
			return nil
		}
		return handleAdvisorQuery(c, advisorService, text)
	})

	log.Println("Telegram bot started")
	go b.Start()
	return alerts
}

func handleAdvisorQuery(c tele.Context, adv Advisor, question string) error {
	_ = c.Notify(tele.Typing)

	reply, err := adv.Ask(context.Background(), c.Chat().ID, question)
	if err != nil {
		log.Printf("advisor error for chat %d: %v", c.Chat().ID, err)
		return c.Send("Sorry, I'm having trouble right now. Try /price or /signals for raw data.")
	}

	if len(reply) > 4000 {
		reply = reply[:4000] + "\n\n[truncated]"
	}

	return c.Send(reply)
}

func parseSignalArgs(args []string) (domain.SignalFilter, error) {
	filter := domain.SignalFilter{Limit: 5}

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}

		if strings.HasPrefix(arg, "--risk=") {
			level, err := strconv.Atoi(strings.TrimPrefix(arg, "--risk="))
			if err != nil {
				return domain.SignalFilter{}, err
			}
			risk := domain.RiskLevel(level)
			if !risk.IsValid() {
				return domain.SignalFilter{}, errors.New("risk out of range")
			}
			filter.Risk = &risk
			continue
		}

		if arg == "--risk" {
			if i+1 >= len(args) {
				return domain.SignalFilter{}, errors.New("missing risk value")
			}
			i++
			level, err := strconv.Atoi(args[i])
			if err != nil {
				return domain.SignalFilter{}, err
			}
			risk := domain.RiskLevel(level)
			if !risk.IsValid() {
				return domain.SignalFilter{}, errors.New("risk out of range")
			}
			filter.Risk = &risk
			continue
		}

		if strings.HasPrefix(arg, "--") {
			return domain.SignalFilter{}, errors.New("unknown option")
		}
		if filter.Symbol != "" {
			return domain.SignalFilter{}, errors.New("multiple symbols provided")
		}
		symbol := strings.ToUpper(arg)
		if _, ok := domain.CoinGeckoID[symbol]; !ok {
			return domain.SignalFilter{}, errors.New("unsupported symbol")
		}
		filter.Symbol = symbol
	}

	return filter, nil
}

func formatSignal(s domain.Signal) string {
	return fmt.Sprintf(
		"%s %s %s %s risk %d at %s",
		s.Symbol,
		s.Interval,
		strings.ToUpper(s.Indicator),
		strings.ToUpper(string(s.Direction)),
		s.Risk,
		s.Timestamp.UTC().Format(time.RFC822),
	)
}
