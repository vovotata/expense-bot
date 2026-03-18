package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/fsm"
	"expense-bot/internal/storage"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type Handler struct {
	store    storage.Storage
	fsm      fsm.StateStore
	adminChat int64
}

func New(store storage.Storage, fsmStore fsm.StateStore, adminChat int64) *Handler {
	return &Handler{
		store:     store,
		fsm:       fsmStore,
		adminChat: adminChat,
	}
}

func (h *Handler) Start(b *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser
	dbCtx := context.Background()

	// Upsert user in DB
	_, err := h.store.UpsertUser(dbCtx, &domain.User{
		ID:        user.Id,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	})
	if err != nil {
		slog.Error("failed to upsert user", "error", err, "user_id", user.Id)
	}

	// Initialize FSM state
	state := &fsm.WizardState{
		UserID:       user.Id,
		CurrentStep:  fsm.StepExpenseType,
		StartedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	if err := h.fsm.Set(dbCtx, state); err != nil {
		slog.Error("failed to set FSM state", "error", err, "user_id", user.Id)
		return fmt.Errorf("start: set FSM: %w", err)
	}

	kb := keyboards.ExpenseTypeKeyboard()
	_, err = ctx.EffectiveMessage.Reply(b, "Выберите тип расходника:", &gotgbot.SendMessageOpts{
		ReplyMarkup: kb,
	})
	return err
}
