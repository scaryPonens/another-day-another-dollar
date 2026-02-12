package bot

import "testing"

func TestStartTelegramBotSkipsWithoutToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	StartTelegramBot(nil)
}
