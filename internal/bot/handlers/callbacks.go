package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"expense-bot/internal/domain"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"expense-bot/internal/fsm"
)

// HandleCallback handles only admin inline callbacks (in the admin group chat).
func (h *Handler) HandleCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	data := cq.Data

	defer func() {
		_, _ = cq.Answer(b, nil)
	}()

	switch {
	case strings.HasPrefix(data, "a:"):
		return h.handleAdminCallback(b, ctx, data)
	case strings.HasPrefix(data, "delmail:"):
		return h.handleDeleteMailCallback(b, ctx, strings.TrimPrefix(data, "delmail:"))
	default:
		return nil
	}
}

func (h *Handler) handleAdminCallback(b *gotgbot.Bot, ctx *ext.Context, data string) error {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return nil
	}
	action := parts[1]
	requestIDStr := parts[2]

	var newStatus domain.RequestStatus
	var statusEmoji string
	switch action {
	case "pd":
		newStatus = domain.StatusPaid
		statusEmoji = "💸"
	case "rj":
		newStatus = domain.StatusRejected
		statusEmoji = "❌"
	default:
		return nil
	}

	reqID, err := uuid.Parse(requestIDStr)
	if err != nil {
		slog.Error("invalid request UUID in admin callback", "uuid", requestIDStr, "error", err)
		return nil
	}

	updated, err := h.store.UpdateRequestStatus(context.Background(), reqID, newStatus)
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

	// Send feedback to the user who created the request
	if updated != nil && updated.UserID != 0 {
		var feedbackText string
		switch newStatus {
		case domain.StatusPaid:
			feedbackText = fmt.Sprintf("💸 Ваша заявка <b>#%s</b> оплачена!", requestIDStr[:8])
		case domain.StatusRejected:
			feedbackText = fmt.Sprintf("❌ Ваша заявка <b>#%s</b> отклонена.", requestIDStr[:8])
		}
		if feedbackText != "" {
			_, _ = b.SendMessage(updated.UserID, feedbackText, &gotgbot.SendMessageOpts{
				ParseMode: "HTML",
			})
		}
	}

	slog.Info("admin status change", "action", action, "request_id", requestIDStr, "status", newStatus)
	return nil
}

func (h *Handler) handleDeleteMailCallback(b *gotgbot.Bot, ctx *ext.Context, idStr string) error {
	userID := ctx.EffectiveUser.Id

	var accountID int64
	_, err := fmt.Sscanf(idStr, "%d", &accountID)
	if err != nil {
		slog.Error("invalid email account ID", "id", idStr, "error", err)
		return nil
	}

	err = h.store.DeleteEmailAccount(context.Background(), accountID, userID)
	if err != nil {
		slog.Error("failed to delete email account", "error", err, "id", accountID)
		_, _, _ = ctx.EffectiveMessage.EditText(b, "❌ Ошибка при удалении.", nil)
		return nil
	}

	_, _, _ = ctx.EffectiveMessage.EditText(b, "✅ Почта удалена.", nil)
	return nil
}

// submitRequest saves the request to DB and notifies admin.
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
		_, _ = b.SendMessage(ctx.EffectiveChat.Id,
			"❌ Не удалось отправить заявку. Попробуйте нажать «Отправить» ещё раз.", nil)
		return nil
	}

	_ = h.fsm.Delete(dbCtx, userID)
	h.restoreMenu(b, ctx.EffectiveChat.Id, userID)
	_, _ = b.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Заявка #%s отправлена!", created.ID.String()[:8]), nil)

	return h.notifyAdmin(b, state, user, created.ID.String())
}

func (h *Handler) notifyAdmin(b *gotgbot.Bot, state *fsm.WizardState, user *gotgbot.User, requestID string) error {
	text := FormatAdminNotification(state, user.Username, user.FirstName, requestID)
	kb := AdminRequestKeyboard(requestID)

	var msg *gotgbot.Message
	var err error

	if state.AddressPhoto != "" {
		msg, err = b.SendPhoto(h.adminChat, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendPhotoOpts{
			Caption:     text,
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
		if err != nil {
			// Fallback: try as document
			msg, err = b.SendDocument(h.adminChat, gotgbot.InputFileByID(state.AddressPhoto), &gotgbot.SendDocumentOpts{
				Caption:     text,
				ParseMode:   "HTML",
				ReplyMarkup: kb,
			})
		}
	} else {
		msg, err = b.SendMessage(h.adminChat, text, &gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
	}

	if err != nil {
		slog.Error("failed to notify admin", "error", err)
		return nil
	}
	_ = msg
	return nil
}

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
