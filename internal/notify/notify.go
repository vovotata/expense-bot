package notify

import (
	"fmt"
	"log/slog"

	"expense-bot/internal/domain"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Notifier struct {
	bot         *gotgbot.Bot
	adminChatID int64
}

func New(bot *gotgbot.Bot, adminChatID int64) *Notifier {
	return &Notifier{
		bot:         bot,
		adminChatID: adminChatID,
	}
}

// NotifyNewRequest sends a formatted request notification to the admin chat.
func (n *Notifier) NotifyNewRequest(req *domain.Request, user *domain.User, keyboard gotgbot.InlineKeyboardMarkup) (int64, error) {
	text := FormatRequestNotification(req, user)

	var msg *gotgbot.Message
	var err error

	if req.AddressPhoto != "" {
		msg, err = n.bot.SendPhoto(n.adminChatID, gotgbot.InputFileByID(req.AddressPhoto), &gotgbot.SendPhotoOpts{
			Caption:     text,
			ParseMode:   "HTML",
			ReplyMarkup: keyboard,
		})
	} else {
		msg, err = n.bot.SendMessage(n.adminChatID, text, &gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: keyboard,
		})
	}

	if err != nil {
		slog.Error("failed to send admin notification", "error", err)
		return 0, fmt.Errorf("notify.NewRequest: %w", err)
	}

	return msg.MessageId, nil
}

// NotifyStatusChange updates the admin message when status changes.
func (n *Notifier) NotifyStatusChange(messageID int64, originalText string, newStatus domain.RequestStatus, isCaption bool) error {
	statusEmoji := statusToEmoji(newStatus)
	updatedText := originalText + fmt.Sprintf("\n\n%s Статус: <b>%s</b>", statusEmoji, newStatus.Label())

	if isCaption {
		_, _, err := n.bot.EditMessageCaption(&gotgbot.EditMessageCaptionOpts{
			ChatId:    n.adminChatID,
			MessageId: messageID,
			Caption:   updatedText,
			ParseMode: "HTML",
		})
		return err
	}

	_, _, err := n.bot.EditMessageText(updatedText, &gotgbot.EditMessageTextOpts{
		ChatId:    n.adminChatID,
		MessageId: messageID,
		ParseMode: "HTML",
	})
	return err
}

// NotifyUser sends a notification to the user about status change.
func (n *Notifier) NotifyUser(chatID int64, requestID string, newStatus domain.RequestStatus) error {
	emoji := statusToEmoji(newStatus)
	text := fmt.Sprintf("%s Заявка #%s: <b>%s</b>", emoji, requestID[:8], newStatus.Label())

	_, err := n.bot.SendMessage(chatID, text, &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

func FormatRequestNotification(req *domain.Request, user *domain.User) string {
	s := fmt.Sprintf("🆕 <b>Заявка #%s</b>\n\n", req.ID.String()[:8])

	if user.Username != "" {
		s += fmt.Sprintf("👤 @%s (%s)\n", user.Username, user.FirstName)
	} else {
		s += fmt.Sprintf("👤 %s\n", user.FirstName)
	}

	s += fmt.Sprintf("📦 Тип: %s\n", req.ExpenseType.Label())

	if req.FlowType == "A" {
		s += fmt.Sprintf("💳 Оплата: %s\n", req.PaymentMethod.Label())
		if req.Address != "" {
			s += fmt.Sprintf("📍 Адрес: <code>%s</code>\n", req.Address)
		}
		if req.AddressPhoto != "" {
			s += "📍 Адрес: [см. фото]\n"
		}
		s += fmt.Sprintf("💰 Сумма: %s\n", req.Amount.String())
	} else {
		s += fmt.Sprintf("🖥 Аккаунт: %s\n", req.AntiqueAccount)
	}

	s += fmt.Sprintf("💬 Комментарий: %s\n", req.Comment)
	s += fmt.Sprintf("\n🕐 Создана: %s", req.CreatedAt.Format("2006-01-02 15:04 MST"))
	return s
}

func statusToEmoji(s domain.RequestStatus) string {
	switch s {
	case domain.StatusApproved:
		return "✅"
	case domain.StatusPaid:
		return "💰"
	case domain.StatusRejected:
		return "❌"
	case domain.StatusCancelled:
		return "🚫"
	default:
		return "🕐"
	}
}
