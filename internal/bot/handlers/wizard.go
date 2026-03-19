package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/fsm"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// HandleTextMessage routes all text: menu buttons, wizard input, wizard buttons.
func (h *Handler) HandleTextMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	text := ctx.EffectiveMessage.Text
	userID := ctx.EffectiveUser.Id

	// Menu buttons (always available)
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
	}

	// Check FSM state
	dbCtx := context.Background()
	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil {
		return fmt.Errorf("wizard.text: get FSM: %w", err)
	}
	if state == nil {
		return nil // no active wizard
	}

	// Global wizard buttons
	switch text {
	case keyboards.BtnCancel:
		return h.cancelWizard(b, ctx, userID)
	case keyboards.BtnBack:
		return h.goBack(b, ctx, state)
	}

	// Route by current step
	switch state.CurrentStep {
	case fsm.StepExpenseType:
		return h.handleExpenseTypeText(b, ctx, state, text)
	case fsm.StepPaymentMethod:
		return h.handlePaymentMethodText(b, ctx, state, text)
	case fsm.StepAddress:
		return h.handleAddressText(b, ctx, state, text)
	case fsm.StepAmount:
		return h.handleAmountText(b, ctx, state, text)
	case fsm.StepAntiqueAccount:
		return h.handleAccountText(b, ctx, state, text)
	case fsm.StepComment:
		return h.handleCommentText(b, ctx, state, text)
	case fsm.StepConfirm:
		return h.handleConfirmText(b, ctx, state, text)
	// Email wizard
	case fsm.StepEmailProvider:
		return h.handleEmailProviderText(b, ctx, state, text)
	case fsm.StepEmailAddress:
		return h.handleEmailAddressText(b, ctx, state, text)
	case fsm.StepEmailPassword:
		return h.handleEmailPasswordText(b, ctx, state, text)
	default:
		return nil
	}
}

func (h *Handler) handleExpenseTypeText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	var expType domain.ExpenseType
	switch text {
	case keyboards.BtnAgentki:
		expType = domain.ExpenseAgentki
	case keyboards.BtnAdpos:
		expType = domain.ExpenseAdpos
	case keyboards.BtnAntique:
		expType = domain.ExpenseAntiqueService
	case keyboards.BtnOther:
		expType = domain.ExpenseOtherService
	case keyboards.BtnSetups:
		expType = domain.ExpenseSetups
	default:
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "Выберите тип из кнопок ниже.", nil)
		return nil
	}

	state.ExpenseType = expType
	if expType == domain.ExpenseAntiqueService {
		state.FlowType = "B"
	} else {
		state.FlowType = "A"
	}
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handlePaymentMethodText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	var pm domain.PaymentMethod
	switch text {
	case keyboards.BtnUSDT:
		pm = domain.PaymentUSDT
	case keyboards.BtnTRX:
		pm = domain.PaymentTRX
	case keyboards.BtnCard:
		pm = domain.PaymentCard
	default:
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "Выберите способ оплаты из кнопок ниже.", nil)
		return nil
	}

	state.PaymentMethod = pm
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handleAddressText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	validate := ChooseAddressValidator(state.PaymentMethod)
	addr, err := validate(text)
	if err != nil {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ "+err.Error(), nil)
		return nil
	}
	state.Address = addr
	state.AddressPhoto = ""
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handleAmountText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	amt, err := ValidateAmount(text)
	if err != nil {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ "+err.Error(), nil)
		return nil
	}
	state.Amount = amt
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handleAccountText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	acct, err := ValidateAccount(text)
	if err != nil {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ "+err.Error(), nil)
		return nil
	}
	state.AntiqueAcct = acct
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handleCommentText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	if text == keyboards.BtnSkip {
		state.Comment = ""
		state.CurrentStep = state.NextStep()
		return h.saveAndSendStep(b, ctx, state)
	}

	comment, err := ValidateComment(text)
	if err != nil {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ "+err.Error(), nil)
		return nil
	}
	state.Comment = comment
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) handleConfirmText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	switch text {
	case keyboards.BtnSubmit:
		return h.submitRequest(b, ctx, state)
	case keyboards.BtnEdit:
		return h.showEditMenu(b, ctx, state)
	case keyboards.BtnCancel:
		return h.cancelWizard(b, ctx, ctx.EffectiveUser.Id)
	// Edit field buttons
	case keyboards.BtnEditType:
		state.CurrentStep = fsm.StepExpenseType
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditPayment:
		state.CurrentStep = fsm.StepPaymentMethod
		state.Address = ""
		state.AddressPhoto = ""
		state.Amount = ""
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditAddress:
		state.CurrentStep = fsm.StepAddress
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditAmount:
		state.CurrentStep = fsm.StepAmount
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditAccount:
		state.CurrentStep = fsm.StepAntiqueAccount
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditComment:
		state.CurrentStep = fsm.StepComment
		return h.saveAndSendStep(b, ctx, state)
	case keyboards.BtnEditBack:
		state.CurrentStep = fsm.StepConfirm
		return h.saveAndSendStep(b, ctx, state)
	default:
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "Выберите действие из кнопок ниже.", nil)
		return nil
	}
}

func (h *Handler) showEditMenu(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	kb := keyboards.EditFieldKeyboard(state.FlowType)
	_, err := b.SendMessage(ctx.EffectiveChat.Id, "Что хотите изменить?", &gotgbot.SendMessageOpts{
		ReplyMarkup: kb,
	})
	return err
}

// HandlePhoto processes photo input during address step.
func (h *Handler) HandlePhoto(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil {
		return fmt.Errorf("wizard.photo: get FSM: %w", err)
	}
	if state == nil || state.CurrentStep != fsm.StepAddress {
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
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Не удалось получить фото. Попробуйте снова.", nil)
		return nil
	}

	state.AddressPhoto = fileID
	state.Address = ""
	state.CurrentStep = state.NextStep()
	return h.saveAndSendStep(b, ctx, state)
}

// --- helpers ---

func (h *Handler) saveAndSendStep(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	if err := h.fsm.Set(context.Background(), state); err != nil {
		return fmt.Errorf("wizard: set FSM: %w", err)
	}
	return h.sendStepMessage(b, ctx, state)
}

func (h *Handler) goBack(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	prev := state.PrevStep()
	if prev == fsm.StepIdle {
		return h.cancelWizard(b, ctx, ctx.EffectiveUser.Id)
	}
	state.CurrentStep = prev
	return h.saveAndSendStep(b, ctx, state)
}

func (h *Handler) cancelWizard(b *gotgbot.Bot, ctx *ext.Context, userID int64) error {
	_ = h.fsm.Delete(context.Background(), userID)
	h.restoreMenu(b, ctx.EffectiveChat.Id, userID)
	_, err := b.SendMessage(ctx.EffectiveChat.Id, "❌ Заявка отменена.", nil)
	return err
}

func (h *Handler) sendStepMessage(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	chatID := ctx.EffectiveChat.Id

	switch state.CurrentStep {
	case fsm.StepExpenseType:
		kb := keyboards.ExpenseTypeKeyboard()
		_, err := b.SendMessage(chatID, "Выберите тип расходника:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepPaymentMethod:
		kb := keyboards.PaymentMethodKeyboard()
		_, err := b.SendMessage(chatID, "Выберите способ оплаты:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepAddress:
		kb := keyboards.InputKeyboard()
		var prompt string
		if state.PaymentMethod == domain.PaymentCard {
			prompt = "Отправьте реквизиты карты (текстом или скриншотом):"
		} else {
			prompt = fmt.Sprintf("Отправьте адрес %s-кошелька (текстом или фото QR):", state.PaymentMethod.Label())
		}
		_, err := b.SendMessage(chatID, prompt, &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepAmount:
		kb := keyboards.InputKeyboard()
		currency := keyboards.CurrencyLabel(state.PaymentMethod)
		_, err := b.SendMessage(chatID, fmt.Sprintf("Введите сумму в %s (например: 12.452):", currency),
			&gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepAntiqueAccount:
		kb := keyboards.InputKeyboard()
		_, err := b.SendMessage(chatID, "Введите аккаунт в Антике:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepComment:
		kb := keyboards.CommentKeyboard()
		var prompt string
		if state.FlowType == "B" {
			prompt = "Добавьте комментарий (сколько поинтов, за какой срок):"
		} else {
			prompt = "Добавьте комментарий (срок оплаты, детали):"
		}
		_, err := b.SendMessage(chatID, prompt, &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepConfirm:
		return h.sendSummary(b, chatID, state)

	// Email wizard steps
	case fsm.StepEmailProvider:
		kb := keyboards.EmailProviderKeyboard()
		_, err := b.SendMessage(chatID, "Выберите почтовый сервис:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepEmailAddress:
		kb := keyboards.EmailInputKeyboard()
		_, err := b.SendMessage(chatID, "Введите email-адрес:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err

	case fsm.StepEmailPassword:
		kb := keyboards.EmailInputKeyboard()
		hint := passwordHint(state.EmailProvider)
		_, err := b.SendMessage(chatID, fmt.Sprintf("Введите пароль приложения:\n\n%s", hint),
			&gotgbot.SendMessageOpts{ReplyMarkup: kb, ParseMode: "HTML"})
		return err

	default:
		return nil
	}
}

func (h *Handler) sendSummary(b *gotgbot.Bot, chatID int64, state *fsm.WizardState) error {
	summary := formatSummary(state)
	kb := keyboards.ConfirmKeyboard()

	if state.AddressPhoto != "" {
		// Try sendPhoto first, fall back to sendDocument if file is a document
		_, err := b.SendPhoto(chatID, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendPhotoOpts{
			Caption:   summary,
			ParseMode: "HTML",
		})
		if err != nil {
			// Fallback: send as document (user may have sent file instead of photo)
			_, err = b.SendDocument(chatID, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendDocumentOpts{
				Caption:   summary,
				ParseMode: "HTML",
			})
			if err != nil {
				// Last resort: just send text summary without the image
				slog.Warn("failed to send address photo in summary", "error", err)
			}
		}
		_, err = b.SendMessage(chatID, "Выберите действие:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err
	}

	_, err := b.SendMessage(chatID, summary, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func formatSummary(state *fsm.WizardState) string {
	s := fmt.Sprintf("📋 <b>Ваша заявка:</b>\n\n• Тип: %s\n", state.ExpenseType.Label())

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

func FormatAdminNotification(state *fsm.WizardState, username, firstName string, requestID string) string {
	s := fmt.Sprintf("🆕 <b>Заявка #%s</b>\n\n", requestID[:8])
	if username != "" {
		s += fmt.Sprintf("👤 @%s (%s)\n", username, firstName)
	} else {
		s += fmt.Sprintf("👤 %s\n", firstName)
	}
	s += fmt.Sprintf("📦 Тип: %s\n", state.ExpenseType.Label())

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

	msk := time.FixedZone("MSK", 3*60*60)
	s += fmt.Sprintf("\n🕐 Получено: %s МСК", time.Now().In(msk).Format("15:04 (02 янв)"))
	return s
}
