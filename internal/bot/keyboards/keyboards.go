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
	BtnMyMails    = "📧 Мои почты"
	BtnAddMail    = "➕ Добавить почту"
	BtnDelMail    = "🗑 Удалить почту"
	BtnCancel     = "❌ Отмена"
	BtnBack       = "⬅️ Назад"
	BtnSkip       = "⏭ Пропустить"
	BtnSubmit     = "✅ Отправить заявку"
	BtnEdit       = "✏️ Редактировать"
)

// Expense type button labels
const (
	BtnAgentki  = "Агентки"
	BtnAdpos    = "Адпос"
	BtnAntique  = "Сервис в Антике"
	BtnOther    = "Другие сервисы"
	BtnSetups   = "Сетапы"
)

// Payment method button labels
const (
	BtnUSDT = "USDT"
	BtnTRX  = "TRX"
	BtnCard = "Карта"
)

// Edit field button labels
const (
	BtnEditType    = "Тип расходника"
	BtnEditPayment = "Способ оплаты"
	BtnEditAddress = "Адрес/Реквизиты"
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
		[]string{BtnCodes, BtnMyMails},
		[]string{BtnAddMail, BtnDelMail},
	)
}

// --- Wizard step keyboards ---

func ExpenseTypeKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{BtnAgentki, BtnAdpos},
		[]string{BtnAntique, BtnOther},
		[]string{BtnSetups},
		[]string{BtnCancel},
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
		[]string{BtnEdit},
		[]string{BtnCancel},
	)
}

func EditFieldKeyboard(flowType string) gotgbot.ReplyKeyboardMarkup {
	if flowType == "A" {
		return reply(
			[]string{BtnEditType},
			[]string{BtnEditPayment, BtnEditAddress},
			[]string{BtnEditAmount, BtnEditComment},
			[]string{BtnEditBack},
		)
	}
	return reply(
		[]string{BtnEditType},
		[]string{BtnEditAccount, BtnEditComment},
		[]string{BtnEditBack},
	)
}

func ConfirmOverwriteKeyboard() gotgbot.ReplyKeyboardMarkup {
	return reply(
		[]string{"Да, начать новую"},
		[]string{"Нет, продолжить"},
	)
}

// --- Admin inline keyboards (these stay inline — they're in the admin GROUP chat) ---

func AdminRequestKeyboard(requestID string) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "💸 Оплачено", CallbackData: fmt.Sprintf("a:pd:%s", requestID)},
			},
			{
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("a:rj:%s", requestID)},
			},
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

type EmailAccountInfo struct {
	ID    int64
	Email string
}
