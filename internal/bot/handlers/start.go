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
	store         storage.Storage
	fsm           fsm.StateStore
	adminChat     int64
	adminUsers    map[int64]struct{}
	encryptionKey string
}

func New(store storage.Storage, fsmStore fsm.StateStore, adminChat int64, adminUserIDs []int64, encryptionKey string) *Handler {
	admins := make(map[int64]struct{}, len(adminUserIDs))
	for _, id := range adminUserIDs {
		admins[id] = struct{}{}
	}
	return &Handler{
		store:         store,
		fsm:           fsmStore,
		adminChat:     adminChat,
		adminUsers:    admins,
		encryptionKey: encryptionKey,
	}
}

func (h *Handler) isAdmin(userID int64) bool {
	_, ok := h.adminUsers[userID]
	return ok
}

// menuKeyboard returns the appropriate persistent keyboard for the user.
func (h *Handler) menuKeyboard(userID int64) gotgbot.ReplyKeyboardMarkup {
	if h.isAdmin(userID) {
		return keyboards.AdminMenuKeyboard()
	}
	return keyboards.UserMenuKeyboard()
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

	// Check if there's an active session — just overwrite, user explicitly clicked "Новая заявка"
	existing, _ := h.fsm.Get(dbCtx, user.Id)
	if existing != nil && existing.CurrentStep != fsm.StepIdle {
		_ = h.fsm.Delete(dbCtx, user.Id)
	}

	return h.startWizard(b, ctx)
}

func (h *Handler) startWizard(b *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser
	dbCtx := context.Background()

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
	_, err := b.SendMessage(ctx.EffectiveChat.Id,
		"Заполните заявку на оплату расходов.\n\nВыберите тип расходника:",
		&gotgbot.SendMessageOpts{ReplyMarkup: kb},
	)
	return err
}

// SendMainMenu sends the persistent ReplyKeyboard menu.
func (h *Handler) SendMainMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser
	dbCtx := context.Background()

	// Upsert user in DB
	_, _ = h.store.UpsertUser(dbCtx, &domain.User{
		ID:        user.Id,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	})

	kb := h.menuKeyboard(user.Id)
	_, err := b.SendMessage(ctx.EffectiveChat.Id,
		"Выберите действие:",
		&gotgbot.SendMessageOpts{ReplyMarkup: kb},
	)
	return err
}

func (h *Handler) getEncryptionKey() string {
	return h.encryptionKey
}

// restoreMenu sends back the persistent menu after wizard completes.
func (h *Handler) restoreMenu(b *gotgbot.Bot, chatID int64, userID int64) {
	kb := h.menuKeyboard(userID)
	_, _ = b.SendMessage(chatID, "👇", &gotgbot.SendMessageOpts{
		ReplyMarkup: kb,
	})
}
