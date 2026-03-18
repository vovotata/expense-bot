package keyboards

import (
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

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

func ConfirmKeyboard() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "✅ Отправить", CallbackData: "confirm:yes"},
				{Text: "✏️ Редактировать", CallbackData: "confirm:edit"},
				{Text: "❌ Отменить", CallbackData: "confirm:cancel"},
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

func AdminRequestKeyboard(requestID string) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "✅ Одобрить", CallbackData: fmt.Sprintf("adm:ap:%s", requestID)},
				{Text: "💰 Оплачено", CallbackData: fmt.Sprintf("adm:pd:%s", requestID)},
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("adm:rj:%s", requestID)},
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

type EmailAccountInfo struct {
	ID    int64
	Email string
}
