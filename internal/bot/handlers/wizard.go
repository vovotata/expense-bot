package handlers

import (
	"context"
	"fmt"
	"time"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/fsm"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// HandleTextMessage processes text input during wizard steps,
// and also routes persistent keyboard button presses.
func (h *Handler) HandleTextMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	text := ctx.EffectiveMessage.Text

	// Route persistent keyboard buttons
	switch text {
	case keyboards.BtnNewRequest:
		return h.Start(b, ctx)
	case keyboards.BtnCodes:
		return h.HandleCodes(b, ctx)
	case keyboards.BtnMyMails:
		return h.HandleMyMails(b, ctx)
	case keyboards.BtnAddMail:
		return h.HandleAddMail(b, ctx)
	case keyboards.BtnDelMail:
		return h.HandleDelMail(b, ctx)
	case keyboards.BtnCancel:
		return h.handleCancelFromKeyboard(b, ctx)
	}

	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil {
		return fmt.Errorf("wizard.text: get FSM: %w", err)
	}
	if state == nil {
		return nil // no active wizard
	}

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
			prompt = fmt.Sprintf("Отправьте адрес %s-кошелька (текстом или фото QR-кода):", state.PaymentMethod.Label())
		}
		_, err := b.SendMessage(chat.Id, prompt, nil)
		return err

	case fsm.StepAmount:
		currency := keyboards.CurrencyLabel(state.PaymentMethod)
		prompt := fmt.Sprintf("Введите сумму в %s (например: 12.452):", currency)
		_, err := b.SendMessage(chat.Id, prompt, nil)
		return err

	case fsm.StepAntiqueAccount:
		_, err := b.SendMessage(chat.Id, "Введите аккаунт в Антике:", nil)
		return err

	case fsm.StepComment:
		var prompt string
		if state.FlowType == "B" {
			prompt = "Добавьте комментарий (сколько осталось поинтов, за какой срок пополнить):"
		} else {
			prompt = "Добавьте комментарий (срок оплаты, детали):"
		}
		kb := keyboards.CommentSkipKeyboard()
		_, err := b.SendMessage(chat.Id, prompt, &gotgbot.SendMessageOpts{
			ReplyMarkup: kb,
		})
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

func (h *Handler) handleCancelFromKeyboard(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	_ = h.fsm.Delete(context.Background(), userID)
	h.restoreMenu(b, ctx.EffectiveChat.Id, userID)
	_, err := b.SendMessage(ctx.EffectiveChat.Id, "❌ Заявка отменена.", nil)
	return err
}

func formatSummary(state *fsm.WizardState) string {
	expType := state.ExpenseType.Label()
	s := fmt.Sprintf("📋 <b>Ваша заявка:</b>\n\n• Тип: %s\n", expType)

	if state.FlowType == "A" {
		currency := keyboards.CurrencyLabel(state.PaymentMethod)
		s += fmt.Sprintf("• Оплата: %s\n", state.PaymentMethod.Label())
		if state.Address != "" {
			s += fmt.Sprintf("• Адрес: <code>%s</code>\n", state.Address)
		}
		if state.AddressPhoto != "" {
			s += "• Адрес: [фото выше]\n"
		}
		s += fmt.Sprintf("• Сумма: %s %s\n", state.Amount, currency)
	} else {
		s += fmt.Sprintf("• Аккаунт: %s\n", state.AntiqueAcct)
	}

	if state.Comment != "" {
		s += fmt.Sprintf("• Комментарий: %s\n", state.Comment)
	}

	s += "\nВсё верно?"
	return s
}

// FormatAdminNotification formats the request for admin chat with timestamp.
func FormatAdminNotification(state *fsm.WizardState, username, firstName string, requestID string) string {
	expType := state.ExpenseType.Label()

	s := fmt.Sprintf("🆕 <b>Заявка #%s</b>\n\n", requestID[:8])
	if username != "" {
		s += fmt.Sprintf("👤 @%s (%s)\n", username, firstName)
	} else {
		s += fmt.Sprintf("👤 %s\n", firstName)
	}
	s += fmt.Sprintf("📦 Тип: %s\n", expType)

	if state.FlowType == "A" {
		currency := keyboards.CurrencyLabel(state.PaymentMethod)
		s += fmt.Sprintf("💳 Оплата: %s\n", state.PaymentMethod.Label())
		if state.Address != "" {
			s += fmt.Sprintf("📍 Адрес: <code>%s</code>\n", state.Address)
		}
		if state.AddressPhoto != "" {
			s += "📍 Адрес: [см. фото]\n"
		}
		s += fmt.Sprintf("💰 Сумма: %s %s\n", state.Amount, currency)
	} else {
		s += fmt.Sprintf("🖥 Аккаунт: %s\n", state.AntiqueAcct)
	}

	if state.Comment != "" {
		s += fmt.Sprintf("💬 Комментарий: %s\n", state.Comment)
	}

	s += fmt.Sprintf("\n🕐 Получено: %s", time.Now().Format("15:04 (02 янв)"))
	return s
}
