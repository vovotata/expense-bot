package handlers

import (
	"context"
	"fmt"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/fsm"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// HandleTextMessage processes text input during wizard steps.
func (h *Handler) HandleTextMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil {
		return fmt.Errorf("wizard.text: get FSM: %w", err)
	}
	if state == nil {
		return nil // no active wizard
	}

	text := ctx.EffectiveMessage.Text

	switch state.CurrentStep {
	case fsm.StepAddress:
		addr, err := ValidateAddress(text)
		if err != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "❌ "+err.Error(), nil)
			return nil
		}
		state.Address = addr
		state.AddressPhoto = ""

	case fsm.StepAmount:
		amt, err := ValidateAmount(text)
		if err != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "❌ "+err.Error(), nil)
			return nil
		}
		state.Amount = amt

	case fsm.StepAntiqueAccount:
		acct, err := ValidateAccount(text)
		if err != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "❌ "+err.Error(), nil)
			return nil
		}
		state.AntiqueAcct = acct

	case fsm.StepComment:
		comment, err := ValidateComment(text)
		if err != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "❌ "+err.Error(), nil)
			return nil
		}
		state.Comment = comment

	default:
		return nil // not expecting text at this step
	}

	// Advance to next step
	state.CurrentStep = state.NextStep()
	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("wizard.text: set FSM: %w", err)
	}

	return h.sendStepMessage(b, ctx, state)
}

// HandlePhoto processes photo input during address step.
func (h *Handler) HandlePhoto(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil {
		return fmt.Errorf("wizard.photo: get FSM: %w", err)
	}
	if state == nil {
		return nil
	}

	if state.CurrentStep != fsm.StepAddress {
		return nil
	}

	msg := ctx.EffectiveMessage
	var fileID string

	if len(msg.Photo) > 0 {
		// Take the largest photo
		fileID = msg.Photo[len(msg.Photo)-1].FileId
	} else if msg.Document != nil {
		fileID = msg.Document.FileId
	}

	if fileID == "" {
		_, _ = msg.Reply(b, "❌ Не удалось получить фото. Попробуйте снова.", nil)
		return nil
	}

	state.AddressPhoto = fileID
	state.Address = ""
	state.CurrentStep = state.NextStep()

	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("wizard.photo: set FSM: %w", err)
	}

	return h.sendStepMessage(b, ctx, state)
}

// sendStepMessage sends the appropriate prompt for the current step.
func (h *Handler) sendStepMessage(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	chat := ctx.EffectiveChat

	switch state.CurrentStep {
	case fsm.StepExpenseType:
		kb := keyboards.ExpenseTypeKeyboard()
		_, err := b.SendMessage(chat.Id, "Выберите тип расходника:", &gotgbot.SendMessageOpts{
			ReplyMarkup: kb,
		})
		return err

	case fsm.StepPaymentMethod:
		kb := keyboards.PaymentMethodKeyboard()
		_, err := b.SendMessage(chat.Id, "Выберите способ оплаты:", &gotgbot.SendMessageOpts{
			ReplyMarkup: kb,
		})
		return err

	case fsm.StepAddress:
		var prompt string
		if state.PaymentMethod == domain.PaymentCard {
			prompt = "Отправьте реквизиты карты / номер карты (текстом или скриншотом):"
		} else {
			prompt = "Отправьте адрес кошелька (текстом или фото QR-кода):"
		}
		_, err := b.SendMessage(chat.Id, prompt, nil)
		return err

	case fsm.StepAmount:
		_, err := b.SendMessage(chat.Id, "Введите точную сумму (например: 12.452):", nil)
		return err

	case fsm.StepAntiqueAccount:
		_, err := b.SendMessage(chat.Id, "Введите аккаунт в Антике:", nil)
		return err

	case fsm.StepComment:
		var prompt string
		if state.FlowType == "B" {
			prompt = "Добавьте комментарий (сколько осталось поинтов, за какой срок пополнить и т.д.):"
		} else {
			prompt = "Добавьте комментарий (срок оплаты, детали):"
		}
		_, err := b.SendMessage(chat.Id, prompt, nil)
		return err

	case fsm.StepConfirm:
		return h.sendSummary(b, chat.Id, state)

	default:
		return nil
	}
}

// sendSummary shows the request summary for confirmation.
func (h *Handler) sendSummary(b *gotgbot.Bot, chatID int64, state *fsm.WizardState) error {
	summary := formatSummary(state)
	kb := keyboards.ConfirmKeyboard()

	if state.AddressPhoto != "" {
		_, err := b.SendPhoto(chatID, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendPhotoOpts{
			Caption:     summary,
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
		return err
	}

	_, err := b.SendMessage(chatID, summary, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func formatSummary(state *fsm.WizardState) string {
	expType := domain.ExpenseType(state.ExpenseType).Label()
	s := fmt.Sprintf("📋 <b>Ваша заявка:</b>\n\n• Тип: %s\n", expType)

	if state.FlowType == "A" {
		s += fmt.Sprintf("• Оплата: %s\n", domain.PaymentMethod(state.PaymentMethod).Label())
		if state.Address != "" {
			s += fmt.Sprintf("• Адрес: %s\n", state.Address)
		}
		if state.AddressPhoto != "" {
			s += "• Адрес: [фото выше]\n"
		}
		s += fmt.Sprintf("• Сумма: %s\n", state.Amount)
	} else {
		s += fmt.Sprintf("• Аккаунт: %s\n", state.AntiqueAcct)
	}

	s += fmt.Sprintf("• Комментарий: %s\n", state.Comment)
	s += "\nВсё верно?"
	return s
}

// FormatAdminNotification formats the request for admin chat.
func FormatAdminNotification(state *fsm.WizardState, username, firstName string, requestID string) string {
	expType := domain.ExpenseType(state.ExpenseType).Label()

	s := fmt.Sprintf("🆕 <b>Заявка #%s</b>\n\n", requestID[:8])
	if username != "" {
		s += fmt.Sprintf("👤 @%s (%s)\n", username, firstName)
	} else {
		s += fmt.Sprintf("👤 %s\n", firstName)
	}
	s += fmt.Sprintf("📦 Тип: %s\n", expType)

	if state.FlowType == "A" {
		s += fmt.Sprintf("💳 Оплата: %s\n", domain.PaymentMethod(state.PaymentMethod).Label())
		if state.Address != "" {
			s += fmt.Sprintf("📍 Адрес: <code>%s</code>\n", state.Address)
		}
		if state.AddressPhoto != "" {
			s += "📍 Адрес: [см. фото]\n"
		}
		s += fmt.Sprintf("💰 Сумма: %s\n", state.Amount)
	} else {
		s += fmt.Sprintf("🖥 Аккаунт: %s\n", state.AntiqueAcct)
	}

	s += fmt.Sprintf("💬 Комментарий: %s\n", state.Comment)
	return s
}
