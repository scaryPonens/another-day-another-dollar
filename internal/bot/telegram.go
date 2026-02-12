package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"bug-free-umbrella/internal/domain"
	"bug-free-umbrella/internal/service"

	tele "gopkg.in/telebot.v3"
)

func StartTelegramBot(priceService *service.PriceService) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Println("TELEGRAM_BOT_TOKEN not set, skipping Telegram bot startup")
		return
	}
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %v", err)
	}

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

	log.Println("Telegram bot started")
	go b.Start()
}
