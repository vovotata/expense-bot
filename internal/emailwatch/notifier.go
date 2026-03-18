package emailwatch

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// TelegramNotifier sends parsed email codes to users via Telegram.
type TelegramNotifier struct {
	bot *gotgbot.Bot
}

// NewTelegramNotifier creates a new Telegram notifier.
func NewTelegramNotifier(bot *gotgbot.Bot) *TelegramNotifier {
	return &TelegramNotifier{bot: bot}
}

// NotifyCode sends a code notification to the user's Telegram chat.
func (n *TelegramNotifier) NotifyCode(chatID int64, email string, result *ParseResult, receivedAt time.Time) (int64, error) {
	text := fmt.Sprintf(
		"🔑 <b>Код подтверждения</b>\n\n"+
			"📧 От: %s\n"+
			"📋 Тема: %s\n"+
			"🔢 Код: <code>%s</code>\n"+
			"⏱ Получен: %s\n\n"+
			"⚠️ Код может быть действителен ограниченное время!",
		result.Sender,
		result.Subject,
		result.Code,
		receivedAt.Format("15:04:05"),
	)

	msg, err := n.bot.SendMessage(chatID, text, &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	if err != nil {
		slog.Error("failed to send code notification",
			"error", err,
			"chat_id", chatID,
			"email", email,
		)
		return 0, fmt.Errorf("notifier.NotifyCode: %w", err)
	}

	return msg.MessageId, nil
}
