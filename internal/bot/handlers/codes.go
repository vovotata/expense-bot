package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"expense-bot/internal/bot/keyboards"
	"expense-bot/internal/domain"
	"expense-bot/internal/emailwatch"
	"expense-bot/internal/fsm"
)

// HandleAddMail starts email wizard (admin only).
func (h *Handler) HandleAddMail(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	if !h.isAdmin(userID) {
		return nil
	}
	dbCtx := context.Background()

	count, err := h.store.CountEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to count email accounts", "error", err)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if count >= 5 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Максимум 5 почтовых ящиков.", nil)
		return nil
	}

	// Start email wizard
	state := &fsm.WizardState{
		UserID:       userID,
		CurrentStep:  fsm.StepEmailProvider,
		FlowType:     "email",
		StartedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("addmail: set FSM: %w", err)
	}

	kb := keyboards.EmailProviderKeyboard()
	_, err = b.SendMessage(ctx.EffectiveChat.Id,
		"Выберите почтовый сервис:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
	return err
}

// handleEmailProviderText handles email provider selection.
func (h *Handler) handleEmailProviderText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	var provider string
	switch text {
	case keyboards.BtnGmail:
		provider = "gmail"
	case keyboards.BtnYandex:
		provider = "yandex"
	case keyboards.BtnMailRu:
		provider = "mailru"
	case keyboards.BtnOutlook:
		provider = "outlook"
	case keyboards.BtnOtherMail:
		provider = "other"
	default:
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "Выберите провайдер из кнопок ниже.", nil)
		return nil
	}

	state.EmailProvider = provider
	state.EmailIMAPHost = keyboards.IMAPServerForProvider(provider)
	state.CurrentStep = fsm.StepEmailAddress
	if err := h.fsm.Set(context.Background(), state); err != nil {
		return err
	}

	kb := keyboards.EmailInputKeyboard()
	_, err := b.SendMessage(ctx.EffectiveChat.Id,
		"Введите email-адрес:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
	return err
}

// handleEmailAddressText handles email address input.
func (h *Handler) handleEmailAddressText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	text = strings.TrimSpace(text)
	if !strings.Contains(text, "@") || !strings.Contains(text, ".") {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Некорректный email. Введите полный адрес (например user@gmail.com):", nil)
		return nil
	}

	state.EmailAddress = text

	// If provider is "other" and we don't have IMAP host, ask for it
	if state.EmailIMAPHost == "" {
		state.CurrentStep = fsm.StepEmailPassword // reuse step, ask IMAP first
		if err := h.fsm.Set(context.Background(), state); err != nil {
			return err
		}
		kb := keyboards.EmailInputKeyboard()
		_, err := b.SendMessage(ctx.EffectiveChat.Id,
			"Введите IMAP-сервер (например imap.example.com:993):",
			&gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err
	}

	state.CurrentStep = fsm.StepEmailPassword
	if err := h.fsm.Set(context.Background(), state); err != nil {
		return err
	}

	kb := keyboards.EmailInputKeyboard()
	hint := passwordHint(state.EmailProvider)
	_, err := b.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("Введите пароль приложения:\n\n%s", hint),
		&gotgbot.SendMessageOpts{ReplyMarkup: kb})
	return err
}

// handleEmailPasswordText handles password input and saves the account.
func (h *Handler) handleEmailPasswordText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	text = strings.TrimSpace(text)
	chatID := ctx.EffectiveChat.Id
	userID := ctx.EffectiveUser.Id

	// If "other" provider and no IMAP host yet, this is the IMAP host input
	if state.EmailIMAPHost == "" {
		if !strings.Contains(text, ":") {
			text += ":993"
		}
		state.EmailIMAPHost = text
		if err := h.fsm.Set(context.Background(), state); err != nil {
			return err
		}
		kb := keyboards.EmailInputKeyboard()
		_, err := b.SendMessage(chatID, "Введите пароль:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
		return err
	}

	state.EmailPassword = text

	// Delete the message with password for security
	_, _ = ctx.EffectiveMessage.Delete(b, nil)

	_, _ = b.SendMessage(chatID, "🔄 Проверяю подключение...", nil)

	// Validate IMAP connection
	err := emailwatch.ValidateIMAPConnection(state.EmailIMAPHost, state.EmailAddress, state.EmailPassword)
	if err != nil {
		slog.Warn("IMAP validation failed", "error", err, "email", state.EmailAddress)
		_, _ = b.SendMessage(chatID,
			fmt.Sprintf("❌ Не удалось подключиться:\n<code>%s</code>\n\nПроверьте данные и попробуйте снова.",
				err.Error()),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
		// Go back to password step
		state.EmailPassword = ""
		state.CurrentStep = fsm.StepEmailPassword
		_ = h.fsm.Set(context.Background(), state)
		return nil
	}

	// Encrypt password and save
	cfg := h.getEncryptionKey()
	encrypted, err := emailwatch.Encrypt(state.EmailPassword, cfg)
	if err != nil {
		slog.Error("failed to encrypt password", "error", err)
		_, _ = b.SendMessage(chatID, "❌ Ошибка шифрования. Попробуйте позже.", nil)
		return nil
	}

	_, err = h.store.CreateEmailAccount(context.Background(), &domain.EmailAccount{
		UserID:      userID,
		Email:       state.EmailAddress,
		IMAPServer:  state.EmailIMAPHost,
		PasswordEnc: encrypted,
	})
	if err != nil {
		slog.Error("failed to create email account", "error", err)
		_, _ = b.SendMessage(chatID, "❌ Ошибка сохранения. Возможно, этот ящик уже добавлен.", nil)
	} else {
		_, _ = b.SendMessage(chatID,
			fmt.Sprintf("✅ Почта <b>%s</b> подключена!\n\nКоды подтверждения будут приходить автоматически.",
				state.EmailAddress),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}

	// Clear FSM and restore menu
	_ = h.fsm.Delete(context.Background(), userID)
	h.restoreMenu(b, chatID, userID)
	return nil
}

func passwordHint(provider string) string {
	switch provider {
	case "gmail":
		return "💡 Для Gmail нужен <b>пароль приложения</b>.\nGoogle Account → Безопасность → Пароли приложений"
	case "yandex":
		return "💡 Для Yandex нужен <b>пароль приложения</b>.\nЯндекс ID → Безопасность → Пароли приложений"
	case "mailru":
		return "💡 Для Mail.ru нужен <b>пароль для внешних приложений</b>.\nНастройки → Безопасность → Пароли приложений"
	case "outlook":
		return "💡 Для Outlook нужен пароль учётной записи Microsoft."
	default:
		return ""
	}
}

// HandleDelMail shows the list of email accounts to delete.
func (h *Handler) HandleDelMail(b *gotgbot.Bot, ctx *ext.Context) error {
	if !h.isAdmin(ctx.EffectiveUser.Id) {
		return nil
	}
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	accounts, err := h.store.ListEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to list email accounts", "error", err)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(accounts) == 0 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "У вас нет подключённых почтовых ящиков.", nil)
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
	_, err = b.SendMessage(ctx.EffectiveChat.Id, "Выберите ящик для удаления:", &gotgbot.SendMessageOpts{
		ReplyMarkup: kb,
	})
	return err
}

// HandleMyMails shows the list of connected email accounts.
func (h *Handler) HandleMyMails(b *gotgbot.Bot, ctx *ext.Context) error {
	if !h.isAdmin(ctx.EffectiveUser.Id) {
		return nil
	}
	userID := ctx.EffectiveUser.Id
	dbCtx := context.Background()

	accounts, err := h.store.ListEmailAccountsByUser(dbCtx, userID)
	if err != nil {
		slog.Error("failed to list email accounts", "error", err)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(accounts) == 0 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "У вас нет подключённых почтовых ящиков.", nil)
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

	_, err = b.SendMessage(ctx.EffectiveChat.Id, sb.String(), &gotgbot.SendMessageOpts{
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
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(codes) == 0 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "Нет перехваченных кодов за последние 24 часа.", nil)
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

	_, err = b.SendMessage(ctx.EffectiveChat.Id, sb.String(), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}
