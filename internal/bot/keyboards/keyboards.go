package keyboards

import (
	"fmt"

	"expense-bot/internal/domain"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// Common button labels
const (
	BtnNewRequest = "📋 Новая заявка"
	BtnCodes      = "🔑 Коды"
	BtnMail       = "📧 Почта"
	BtnCancel     = "❌ Отмена"
	BtnBack       = "⬅️ Назад"
	BtnSkip       = "⏭ Пропустить"
	BtnSubmit     = "✅ Отправить заявку"
	BtnEdit       = "✏️ Редактировать"
)

// Mail submenu buttons
const (
	BtnMyMails = "📋 Мои почты"
	BtnAddMail = "➕ Добавить почту"
	BtnDelMail = "🗑 Удалить почту"
	BtnMailBack = "⬅️ Назад в меню"
)

// Expense type button labels
const (
	BtnAgentki = "Агентки"
	BtnAdpos   = "Адпос"
	BtnAntique = "Сервис в Антике"
	BtnOther   = "Другие сервисы"
	BtnSetups  = "Сетапы"
	BtnProxy   = "Прокси"
)

// Payment method button labels
const (
	BtnUSDT = "USDT"
	BtnTRX  = "TRX"
	BtnCard = "Карта"
)

// Agent name button labels (for Агентки)
const (
	BtnCrossGif = "CROSSGIF"
	BtnULD      = "ULD"
	BtnPremium  = "PREMIUM"
)

// Proxy provider button labels
const (
	BtnProxy6      = "Proxy6"
	BtnProxySeller = "ProxySeller"
)

// Edit field button labels
const (
	BtnEditType    = "Тип расходника"
	BtnEditAgent   = "Агентка"
	BtnEditProxy   = "Прокси-сервис"
	BtnEditPayment = "Способ оплаты"
	BtnEditAddress = "Реквизиты"
	BtnEditAmount  = "Сумма"
	BtnEditAccount = "Аккаунт"
	BtnEditComment = "Комментарий"
	BtnEditBack    = "⬅️ Назад к сводке"
)

func reply(rows ...[]string) gotgbot.ReplyKeyboardMarkup {
	var kb [][]gotgbot.KeyboardButton
	for _, row := range rows {
		var btnRow []gotgbot.KeyboardButton
		for _, text := range row {
			btnRow = append(btnRow, gotgbot.KeyboardButton{Text: text})
		}
		kb = append(kb, btnRow)
	}
	return gotgbot.ReplyKeyboardMarkup{
		Keyboard:       kb,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// --- Menu keyboards ---

func UserMenuKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnNewRequest},
		[]string{BtnCodes},
	)
}

func AdminMenuKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnNewRequest},
		[]string{BtnCodes, BtnMail},
	)
}

// MailSubmenuKeyboard — admin only, shown when "📧 Почта" is pressed.
func MailSubmenuKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnMyMails},
		[]string{BtnAddMail, BtnDelMail},
		[]string{BtnMailBack},
	)
}

// --- Wizard step keyboards ---

func ExpenseTypeKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnAgentki, BtnAdpos},
		[]string{BtnSetups, BtnProxy},
		[]string{BtnOther},
		[]string{BtnAntique},
		[]string{BtnCancel},
	)
}

func AgentNameKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnCrossGif, BtnULD, BtnPremium},
		[]string{BtnBack, BtnCancel},
	)
}

func ProxyProviderKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnProxy6, BtnProxySeller},
		[]string{BtnBack, BtnCancel},
	)
}

func PaymentMethodKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnUSDT, BtnTRX, BtnCard},
		[]string{BtnBack, BtnCancel},
	)
}

func InputKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnBack, BtnCancel},
	)
}

func CommentKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnSkip},
		[]string{BtnBack, BtnCancel},
	)
}

func ConfirmKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnSubmit},
		[]string{BtnEdit, BtnCancel},
	)
}

func EditFieldKeyboard(flowType string, hasAgentName bool, hasProxyProvider bool) gotgbot.ReplyKeyboardMarkup {
	if flowType == "A" {
		rows := [][]string{{BtnEditType}}
		if hasAgentName {
			rows = append(rows, []string{BtnEditAgent})
		}
		if hasProxyProvider {
			rows = append(rows, []string{BtnEditProxy})
		}
		rows = append(rows,
			[]string{BtnEditPayment},
			[]string{BtnEditAddress},
			[]string{BtnEditAmount, BtnEditComment},
			[]string{BtnEditBack},
			[]string{BtnCancel},
		)
		return reply(rows...)
	}
	return reply(
		[]string{BtnEditType},
		[]string{BtnEditAccount},
		[]string{BtnEditComment},
		[]string{BtnEditBack},
		[]string{BtnCancel},
	)
}

// --- Admin inline keyboards (in admin group chat) ---

func AdminRequestKeyboard(requestID string) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "💸 Оплачено", CallbackData: fmt.Sprintf("a:pd:%s", requestID)}},
			{{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("a:rj:%s", requestID)}},
		},
	}
}

func EmailAccountsKeyboard(accounts []EmailAccountInfo) gotgbot.InlineKeyboardMarkup {
	var rows [][]gotgbot.InlineKeyboardButton
	for _, acc := range accounts {
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("🗑 %s", acc.Email), CallbackData: fmt.Sprintf("delmail:%d", acc.ID)},
		})
	}
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func CurrencyLabel(pm domain.PaymentMethod) string {
	switch pm {
	case domain.PaymentUSDT:
		return "USDT"
	case domain.PaymentTRX:
		return "TRX"
	case domain.PaymentCard:
		return "₽"
	default:
		return ""
	}
}

func IMAPServerForProvider(provider string) string {
	switch provider {
	case "gmail":
		return "imap.gmail.com:993"
	case "yandex":
		return "imap.yandex.ru:993"
	case "mailru":
		return "imap.mail.ru:993"
	case "outlook":
		return "outlook.office365.com:993"
	default:
		return ""
	}
}

func EmailInputKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnBack, BtnCancel},
	)
}

type EmailAccountInfo struct {
	ID    int64
	Email string
}
