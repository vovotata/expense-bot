package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/fsm"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// HandleCallback routes all callback queries.
func (h *Handler) HandleCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	data := cq.Data

	// Always answer callback to stop spinner
	defer func() {
		_, _ = cq.Answer(b, nil)
	}()

	switch {
	case strings.HasPrefix(data, "type:"):
		return h.handleTypeCallback(b, ctx, strings.TrimPrefix(data, "type:"))
	case strings.HasPrefix(data, "pay:"):
		return h.handlePaymentCallback(b, ctx, strings.TrimPrefix(data, "pay:"))
	case strings.HasPrefix(data, "confirm:"):
		return h.handleConfirmCallback(b, ctx, strings.TrimPrefix(data, "confirm:"))
	case strings.HasPrefix(data, "edit:"):
		return h.handleEditCallback(b, ctx, strings.TrimPrefix(data, "edit:"))
	case strings.HasPrefix(data, "a:"):
		return h.handleAdminCallback(b, ctx, data)
	case strings.HasPrefix(data, "delmail:"):
		return h.handleDeleteMailCallback(b, ctx, strings.TrimPrefix(data, "delmail:"))
	default:
		slog.Warn("unknown callback data", "data", data)
		return nil
	}
}

func (h *Handler) handleTypeCallback(b *gotgbot.Bot, ctx *ext.Context, expType string) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil || state == nil {
		return h.replyNoSession(b, ctx)
	}
	if state.CurrentStep != fsm.StepExpenseType {
		return nil
	}

	state.ExpenseType = domain.ExpenseType(expType)

	if expType == string(domain.ExpenseAntiqueService) {
		state.FlowType = "B"
	} else {
		state.FlowType = "A"
	}

	state.CurrentStep = state.NextStep()
	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("callback.type: set FSM: %w", err)
	}

	// Edit original message to show selection
	editMessageText(b, ctx.EffectiveMessage,
		fmt.Sprintf("Тип расходника: <b>%s</b>", state.ExpenseType.Label()), nil)

	return h.sendStepMessage(b, ctx, state)
}

func (h *Handler) handlePaymentCallback(b *gotgbot.Bot, ctx *ext.Context, payment string) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil || state == nil {
		return h.replyNoSession(b, ctx)
	}
	if state.CurrentStep != fsm.StepPaymentMethod {
		return nil
	}

	state.PaymentMethod = domain.PaymentMethod(payment)
	state.CurrentStep = state.NextStep()

	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("callback.payment: set FSM: %w", err)
	}

	editMessageText(b, ctx.EffectiveMessage,
		fmt.Sprintf("Способ оплаты: <b>%s</b>", state.PaymentMethod.Label()), nil)

	return h.sendStepMessage(b, ctx, state)
}

func (h *Handler) handleConfirmCallback(b *gotgbot.Bot, ctx *ext.Context, action string) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil || state == nil {
		return h.replyNoSession(b, ctx)
	}
	if state.CurrentStep != fsm.StepConfirm {
		return nil
	}

	switch action {
	case "yes":
		return h.submitRequest(b, ctx, state)
	case "edit":
		kb := keyboards.EditFieldKeyboard(state.FlowType)
		// Delete the summary message (especially important for photos) and send new
		_, _ = ctx.EffectiveMessage.Delete(b, nil)
		_, err := b.SendMessage(ctx.EffectiveChat.Id, "Что хотите изменить?", &gotgbot.SendMessageOpts{
			ReplyMarkup: kb,
		})
		return err
	case "cancel":
		// First update the message, then delete FSM state
		_, _ = ctx.EffectiveMessage.Delete(b, nil)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Заявка отменена.", nil)
		_ = h.fsm.Delete(dbCtx, userID)
		return nil
	}
	return nil
}

func (h *Handler) handleEditCallback(b *gotgbot.Bot, ctx *ext.Context, field string) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	state, err := h.fsm.Get(dbCtx, userID)
	if err != nil || state == nil {
		return h.replyNoSession(b, ctx)
	}

	switch field {
	case "type":
		state.CurrentStep = fsm.StepExpenseType
	case "payment":
		state.CurrentStep = fsm.StepPaymentMethod
	case "address":
		state.CurrentStep = fsm.StepAddress
	case "amount":
		state.CurrentStep = fsm.StepAmount
	case "account":
		state.CurrentStep = fsm.StepAntiqueAccount
	case "comment":
		state.CurrentStep = fsm.StepComment
	case "back":
		state.CurrentStep = fsm.StepConfirm
	default:
		return nil
	}

	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("callback.edit: set FSM: %w", err)
	}

	return h.sendStepMessage(b, ctx, state)
}

func (h *Handler) submitRequest(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()
	user := ctx.EffectiveUser

	amt := decimal.Zero
	if state.Amount != "" {
		amt, _ = decimal.NewFromString(state.Amount)
	}

	req := &domain.Request{
		UserID:         userID,
		ExpenseType:    state.ExpenseType,
		PaymentMethod:  state.PaymentMethod,
		Address:        state.Address,
		AddressPhoto:   state.AddressPhoto,
		Amount:         amt,
		AntiqueAccount: state.AntiqueAcct,
		Comment:        state.Comment,
		FlowType:       state.FlowType,
	}

	created, err := h.store.CreateRequest(dbCtx, req)
	if err != nil {
		slog.Error("failed to create request", "error", err, "user_id", userID)
		_, _ = ctx.EffectiveMessage.Delete(b, nil)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка при сохранении заявки. Попробуйте /start заново.", nil)
		return nil
	}

	// Clear FSM
	_ = h.fsm.Delete(dbCtx, userID)

	// Delete summary message and send confirmation
	_, _ = ctx.EffectiveMessage.Delete(b, nil)
	_, _ = b.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Заявка #%s отправлена!", created.ID.String()[:8]), nil)

	// Send notification to admin chat
	return h.notifyAdmin(b, state, user, created.ID.String())
}

func (h *Handler) notifyAdmin(b *gotgbot.Bot, state *fsm.WizardState, user *gotgbot.User, requestID string) error {
	text := FormatAdminNotification(state, user.Username, user.FirstName, requestID)
	kb := keyboards.AdminRequestKeyboard(requestID)

	var msg *gotgbot.Message
	var err error

	if state.AddressPhoto != "" {
		msg, err = b.SendPhoto(h.adminChat, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendPhotoOpts{
			Caption:     text,
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
	} else {
		msg, err = b.SendMessage(h.adminChat, text, &gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
	}

	if err != nil {
		slog.Error("failed to notify admin", "error", err)
		return nil // don't fail the user flow
	}

	_ = msg // tg_message_id could be stored for future message updates
	return nil
}

func (h *Handler) handleAdminCallback(b *gotgbot.Bot, ctx *ext.Context, data string) error {
	// Format: a:ACTION:FULL_UUID
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return nil
	}
	action := parts[1]
	requestIDStr := parts[2]

	var newStatus domain.RequestStatus
	var statusEmoji string
	switch action {
	case "ap":
		newStatus = domain.StatusApproved
		statusEmoji = "✅"
	case "pd":
		newStatus = domain.StatusPaid
		statusEmoji = "💰"
	case "rj":
		newStatus = domain.StatusRejected
		statusEmoji = "❌"
	default:
		return nil
	}

	// Persist status change in database
	reqID, err := uuid.Parse(requestIDStr)
	if err != nil {
		slog.Error("invalid request UUID in admin callback", "uuid", requestIDStr, "error", err)
		return nil
	}

	_, err = h.store.UpdateRequestStatus(context.Background(), reqID, newStatus)
	if err != nil {
		slog.Error("failed to update request status", "error", err, "request_id", requestIDStr)
	}

	// Update admin message
	currentText := ctx.EffectiveMessage.Text
	if currentText == "" {
		currentText = ctx.EffectiveMessage.Caption
	}

	updatedText := currentText + fmt.Sprintf("\n\n%s Статус: <b>%s</b>", statusEmoji, newStatus.Label())

	if ctx.EffectiveMessage.Caption != "" {
		_, _, _ = ctx.EffectiveMessage.EditCaption(b, &gotgbot.EditMessageCaptionOpts{
			Caption:   updatedText,
			ParseMode: "HTML",
		})
	} else {
		_, _, _ = ctx.EffectiveMessage.EditText(b, updatedText, &gotgbot.EditMessageTextOpts{
			ParseMode: "HTML",
		})
	}

	slog.Info("admin status change", "action", action, "request_id", requestIDStr, "status", newStatus)
	return nil
}

func (h *Handler) handleDeleteMailCallback(b *gotgbot.Bot, ctx *ext.Context, idStr string) error {
	editMessageText(b, ctx.EffectiveMessage, "Email удалён.", nil)
	return nil
}

func (h *Handler) replyNoSession(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := b.SendMessage(ctx.EffectiveChat.Id,
		"Сессия истекла. Начните заново: /start", nil)
	return err
}

// editMessageText is a helper that handles both text messages and photo captions.
func editMessageText(b *gotgbot.Bot, msg *gotgbot.Message, text string, replyMarkup *gotgbot.InlineKeyboardMarkup) {
	if msg.Caption != "" || len(msg.Photo) > 0 {
		opts := &gotgbot.EditMessageCaptionOpts{
			Caption:   text,
			ParseMode: "HTML",
		}
		if replyMarkup != nil {
			opts.ReplyMarkup = *replyMarkup
		}
		_, _, _ = msg.EditCaption(b, opts)
	} else {
		opts := &gotgbot.EditMessageTextOpts{
			ParseMode: "HTML",
		}
		if replyMarkup != nil {
			opts.ReplyMarkup = *replyMarkup
		}
		_, _, _ = msg.EditText(b, text, opts)
	}
}
