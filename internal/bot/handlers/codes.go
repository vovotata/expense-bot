package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"expense-bot/internal/bot/keyboards"
)

// HandleAddMail starts the process of adding an email for monitoring.
func (h *Handler) HandleAddMail(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	count, err := h.store.CountEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to count email accounts", "error", err)
		_, _ = ctx.EffectiveMessage.Reply(b, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if count >= 5 {
		_, _ = ctx.EffectiveMessage.Reply(b, "❌ Максимум 5 почтовых ящиков.", nil)
		return nil
	}

	_, err = ctx.EffectiveMessage.Reply(b,
		"Введите данные почты в формате (каждое поле на новой строке):\n\n"+
			"<code>email@example.com\nimap.example.com:993\npassword123</code>",
		&gotgbot.SendMessageOpts{ParseMode: "HTML"},
	)
	return err
}

// HandleDelMail shows the list of email accounts to delete.
func (h *Handler) HandleDelMail(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	accounts, err := h.store.ListEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to list email accounts", "error", err)
		_, _ = ctx.EffectiveMessage.Reply(b, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(accounts) == 0 {
		_, _ = ctx.EffectiveMessage.Reply(b, "У вас нет подключённых почтовых ящиков.", nil)
		return nil
	}

	var infos []keyboards.EmailAccountInfo
	for _, acc := range accounts {
		infos = append(infos, keyboards.EmailAccountInfo{
			ID:    acc.ID,
			Email: acc.Email,
		})
	}

	kb := keyboards.EmailAccountsKeyboard(infos)
	_, err = ctx.EffectiveMessage.Reply(b, "Выберите ящик для удаления:", &gotgbot.SendMessageOpts{
		ReplyMarkup: kb,
	})
	return err
}

// HandleMyMails shows the list of connected email accounts.
func (h *Handler) HandleMyMails(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	accounts, err := h.store.ListEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to list email accounts", "error", err)
		_, _ = ctx.EffectiveMessage.Reply(b, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(accounts) == 0 {
		_, _ = ctx.EffectiveMessage.Reply(b, "У вас нет подключённых почтовых ящиков.", nil)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📧 <b>Ваши почтовые ящики:</b>\n\n")
	for i, acc := range accounts {
		status := "✅"
		if !acc.IsActive {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s (%s)\n", i+1, status, acc.Email, acc.IMAPServer))
		if acc.LastError != nil && *acc.LastError != "" {
			sb.WriteString(fmt.Sprintf("   ⚠️ Ошибка: %s\n", *acc.LastError))
		}
	}

	_, err = ctx.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

// HandleCodes shows the last 10 intercepted codes.
func (h *Handler) HandleCodes(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	codes, err := h.store.ListRecentCodesByUser(dbCtx, userID, 10)
	if err != nil {
		slog.Error("failed to list codes", "error", err)
		_, _ = ctx.EffectiveMessage.Reply(b, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(codes) == 0 {
		_, _ = ctx.EffectiveMessage.Reply(b, "Нет перехваченных кодов за последние 24 часа.", nil)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("🔑 <b>Последние коды:</b>\n\n")
	for _, code := range codes {
		sb.WriteString(fmt.Sprintf("📧 %s\n", code.Email))
		sb.WriteString(fmt.Sprintf("📋 От: %s\n", code.Sender))
		sb.WriteString(fmt.Sprintf("🔢 Код: <code>%s</code>\n", code.Code))
		sb.WriteString(fmt.Sprintf("⏱ %s\n\n", code.ReceivedAt.Format("15:04:05")))
	}

	_, err = ctx.EffectiveMessage.Reply(b, sb.String(), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}
