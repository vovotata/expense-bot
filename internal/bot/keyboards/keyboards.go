package keyboards

import (
	"fmt"

	"expense-bot/internal/domain"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// Button labels for persistent ReplyKeyboard
const (
	BtnNewRequest = "📋 Новая заявка"
	BtnCodes      = "🔑 Коды"
	BtnMyMails    = "📧 Мои почты"
	BtnAddMail    = "➕ Добавить почту"
	BtnDelMail    = "🗑 Удалить почту"
)

// MainMenuKeyboard returns a persistent ReplyKeyboardMarkup for the user.
func MainMenuKeyboard() gotgbot.ReplyKeyboardMarkup {
	return gotgbot.ReplyKeyboardMarkup{
		Keyboard: [][]gotgbot.KeyboardButton{
			{{Text: BtnNewRequest}},
			{{Text: BtnCodes}, {Text: BtnMyMails}},
			{{Text: BtnAddMail}, {Text: BtnDelMail}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

func ExpenseTypeKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "Агентки", CallbackData: "type:agentki"},
				{Text: "Адпос", CallbackData: "type:adpos"},
			},
			{
				{Text: "Сервис в Антике", CallbackData: "type:antique_service"},
				{Text: "Другие сервисы", CallbackData: "type:other_service"},
			},
			{
				{Text: "Сетапы", CallbackData: "type:setups"},
			},
		},
	}
}

func PaymentMethodKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "USDT", CallbackData: "pay:usdt"},
				{Text: "TRX", CallbackData: "pay:trx"},
				{Text: "Карта", CallbackData: "pay:card"},
			},
		},
	}
}

// ConfirmKeyboard — destructive "Отменить" on a separate row from "Отправить".
func ConfirmKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "✅ Отправить заявку", CallbackData: "confirm:yes"},
			},
			{
				{Text: "✏️ Редактировать", CallbackData: "confirm:edit"},
			},
			{
				{Text: "❌ Отменить", CallbackData: "confirm:cancel"},
			},
		},
	}
}

// CommentSkipKeyboard adds a "Пропустить" button for optional comment.
func CommentSkipKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "⏭ Пропустить", CallbackData: "skip:comment"},
			},
		},
	}
}

// ConfirmOverwriteKeyboard asks user to confirm overwriting active session.
func ConfirmOverwriteKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "Да, начать новую", CallbackData: "overwrite:yes"},
				{Text: "Нет, продолжить", CallbackData: "overwrite:no"},
			},
		},
	}
}

func EditFieldKeyboard(flowType string) gotgbot.InlineKeyboardMarkup {
	var rows [][]gotgbot.InlineKeyboardButton

	rows = append(rows, []gotgbot.InlineKeyboardButton{
		{Text: "Тип расходника", CallbackData: "edit:type"},
	})

	if flowType == "A" {
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: "Способ оплаты", CallbackData: "edit:payment"},
			{Text: "Адрес/Реквизиты", CallbackData: "edit:address"},
		})
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: "Сумма", CallbackData: "edit:amount"},
		})
	} else {
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: "Аккаунт", CallbackData: "edit:account"},
		})
	}

	rows = append(rows, []gotgbot.InlineKeyboardButton{
		{Text: "Комментарий", CallbackData: "edit:comment"},
	})
	rows = append(rows, []gotgbot.InlineKeyboardButton{
		{Text: "⬅️ Назад к сводке", CallbackData: "edit:back"},
	})

	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// AdminRequestKeyboard — only "Оплачено" and "Отклонить", on separate rows.
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

// CurrencyLabel returns the display label for amount based on payment method.
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
