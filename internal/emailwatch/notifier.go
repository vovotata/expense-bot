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

// NotifyCode sends a code notification to a single chat.
func (n *TelegramNotifier) NotifyCode(chatID int64, email string, result *ParseResult, receivedAt time.Time) (int64, error) {
	msk := time.FixedZone("MSK", 3*60*60)
	text := fmt.Sprintf(
		"🔑 <b>Код подтверждения</b>\n\n"+
			"📧 Почта: %s\n"+
			"📋 От: %s\n"+
			"📋 Тема: %s\n"+
			"🔢 Код: <code>%s</code>\n"+
			"⏱ Получен: %s МСК\n\n"+
			"⚠️ Код может быть действителен ограниченное время!",
		email,
		result.Sender,
		result.Subject,
		result.Code,
		receivedAt.In(msk).Format("15:04:05"),
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

// BroadcastCode sends a code notification to multiple chat IDs.
func (n *TelegramNotifier) BroadcastCode(chatIDs []int64, email string, result *ParseResult, receivedAt time.Time) int64 {
	var lastMsgID int64
	for _, chatID := range chatIDs {
		msgID, err := n.NotifyCode(chatID, email, result, receivedAt)
		if err != nil {
			slog.Warn("failed to broadcast code to user", "chat_id", chatID, "error", err)
			continue
		}
		lastMsgID = msgID
	}
	return lastMsgID
}
