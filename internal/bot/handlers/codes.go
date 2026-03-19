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

// HandleAddMail starts email wizard — ask email first, auto-detect provider (admin only).
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

	state := &fsm.WizardState{
		UserID:       userID,
		CurrentStep:  fsm.StepEmailAddress,
		FlowType:     "email",
		StartedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	if err := h.fsm.Set(dbCtx, state); err != nil {
		return fmt.Errorf("addmail: set FSM: %w", err)
	}

	kb := keyboards.EmailInputKeyboard()
	_, err = b.SendMessage(ctx.EffectiveChat.Id,
		"Введите email-адрес (например user@gmail.com):",
		&gotgbot.SendMessageOpts{ReplyMarkup: kb})
	return err
}

// detectProvider returns provider name and IMAP server from email domain.
func detectProvider(email string) (provider, imapHost string) {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "other", ""
	}
	domain := strings.ToLower(parts[1])

	switch {
	case domain == "gmail.com" || domain == "googlemail.com":
		return "gmail", "imap.gmail.com:993"
	case domain == "yandex.ru" || domain == "yandex.com" || domain == "ya.ru":
		return "yandex", "imap.yandex.ru:993"
	case domain == "mail.ru" || domain == "inbox.ru" || domain == "list.ru" || domain == "bk.ru":
		return "mailru", "imap.mail.ru:993"
	case domain == "outlook.com" || domain == "hotmail.com" || domain == "live.com":
		return "outlook", "outlook.office365.com:993"
	default:
		return "other", ""
	}
}

// handleEmailAddressText handles email address input + auto-detects provider.
func (h *Handler) handleEmailAddressText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	text = strings.TrimSpace(strings.ToLower(text))
	chatID := ctx.EffectiveChat.Id

	if !strings.Contains(text, "@") || !strings.Contains(text, ".") || len(text) < 5 {
		_, _ = b.SendMessage(chatID, "❌ Некорректный email. Введите полный адрес:", nil)
		return nil
	}

	provider, imapHost := detectProvider(text)
	state.EmailAddress = text
	state.EmailProvider = provider
	state.EmailIMAPHost = imapHost

	if imapHost == "" {
		// Unknown provider — ask for IMAP server
		state.CurrentStep = fsm.StepEmailProvider // reuse for IMAP host input
		if err := h.fsm.Set(context.Background(), state); err != nil {
			return err
		}
		kb := keyboards.EmailInputKeyboard()
		_, err := b.SendMessage(chatID,
			fmt.Sprintf("Провайдер для <b>%s</b> не определён автоматически.\n\nВведите IMAP-сервер (например imap.example.com:993):", text),
			&gotgbot.SendMessageOpts{ReplyMarkup: kb, ParseMode: "HTML"})
		return err
	}

	// Known provider — go to password
	state.CurrentStep = fsm.StepEmailPassword
	if err := h.fsm.Set(context.Background(), state); err != nil {
		return err
	}

	kb := keyboards.EmailInputKeyboard()
	hint := passwordHint(provider)
	_, err := b.SendMessage(chatID,
		fmt.Sprintf("📧 <b>%s</b> (%s)\n\n%s\n\nВведите пароль приложения:\n\n🔒 <i>Сообщение с паролем будет удалено после отправки.</i>", text, providerLabel(provider), hint),
		&gotgbot.SendMessageOpts{ReplyMarkup: kb, ParseMode: "HTML"})
	return err
}

// handleEmailProviderText — for "other" provider, handles IMAP host input.
func (h *Handler) handleEmailProviderText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	text = strings.TrimSpace(text)
	chatID := ctx.EffectiveChat.Id

	if !strings.Contains(text, ".") {
		_, _ = b.SendMessage(chatID, "❌ Некорректный IMAP-сервер. Формат: imap.example.com:993", nil)
		return nil
	}
	if !strings.Contains(text, ":") {
		text += ":993"
	}

	state.EmailIMAPHost = text
	state.CurrentStep = fsm.StepEmailPassword
	if err := h.fsm.Set(context.Background(), state); err != nil {
		return err
	}

	kb := keyboards.EmailInputKeyboard()
	_, err := b.SendMessage(chatID, "Введите пароль:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
	return err
}

// handleEmailPasswordText handles password input, validates IMAP, saves account.
func (h *Handler) handleEmailPasswordText(b *gotgbot.Bot, ctx *ext.Context, state *fsm.WizardState, text string) error {
	text = strings.TrimSpace(text)
	chatID := ctx.EffectiveChat.Id
	userID := ctx.EffectiveUser.Id

	state.EmailPassword = text

	// Delete message with password for security
	_, _ = ctx.EffectiveMessage.Delete(b, nil)

	_, _ = b.SendMessage(chatID, "🔒 Сообщение с паролем удалено.\n🔄 Проверяю подключение к "+state.EmailIMAPHost+"...", nil)

	// Validate IMAP connection
	err := emailwatch.ValidateIMAPConnection(state.EmailIMAPHost, state.EmailAddress, state.EmailPassword)
	if err != nil {
		slog.Warn("IMAP validation failed", "error", err, "email", state.EmailAddress)
		_, _ = b.SendMessage(chatID,
			fmt.Sprintf("❌ Не удалось подключиться:\n<code>%s</code>\n\nПроверьте пароль и попробуйте снова.", err.Error()),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
		state.EmailPassword = ""
		_ = h.fsm.Set(context.Background(), state)
		return nil
	}

	// Encrypt and save
	encrypted, err := emailwatch.Encrypt(state.EmailPassword, h.getEncryptionKey())
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
			fmt.Sprintf("✅ Почта <b>%s</b> подключена!\n\nКоды подтверждения будут приходить автоматически.", state.EmailAddress),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}

	_ = h.fsm.Delete(context.Background(), userID)
	h.restoreMenu(b, chatID, userID)
	return nil
}

func providerLabel(provider string) string {
	switch provider {
	case "gmail":
		return "Gmail"
	case "yandex":
		return "Yandex"
	case "mailru":
		return "Mail.ru"
	case "outlook":
		return "Outlook"
	default:
		return "Email"
	}
}

func passwordHint(provider string) string {
	switch provider {
	case "gmail":
		return "💡 Нужен <b>пароль приложения</b> (не обычный пароль!).\n\n" +
			"1. Включите двухэтапную аутентификацию в Google\n" +
			"2. Перейдите: myaccount.google.com/apppasswords\n" +
			"3. Создайте пароль для «Почта»\n" +
			"4. Скопируйте 16-символьный пароль сюда"
	case "yandex":
		return "💡 Нужен <b>пароль приложения</b>.\n\n" +
			"1. Перейдите: id.yandex.ru/security/app-passwords\n" +
			"2. Создайте пароль для «Почта IMAP»\n" +
			"3. Скопируйте пароль сюда"
	case "mailru":
		return "💡 Нужен <b>пароль для внешних приложений</b>.\n\n" +
			"1. Откройте Настройки Mail.ru → Безопасность\n" +
			"2. Раздел «Пароли для внешних приложений»\n" +
			"3. Создайте пароль и скопируйте сюда"
	case "outlook":
		return "💡 Используйте пароль учётной записи Microsoft.\n" +
			"Если включена 2FA — создайте пароль приложения на account.microsoft.com"
	default:
		return "Введите пароль от почты или пароль приложения (если есть 2FA)."
	}
}

// HandleTestCode creates a fake code in DB for testing (admin only).
func (h *Handler) HandleTestCode(b *gotgbot.Bot, ctx *ext.Context) error {
	if !h.isAdmin(ctx.EffectiveUser.Id) {
		return nil
	}
	dbCtx := context.Background()

	// Get first email account to attach the test code to
	accounts, err := h.store.ListEmailAccountsByUser(dbCtx, ctx.EffectiveUser.Id)
	if err != nil || len(accounts) == 0 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Сначала добавьте почту через «➕ Добавить почту».", nil)
		return nil
	}

	// Generate a random-ish test code
	testCode := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	_, err = h.store.CreateEmailCode(dbCtx, &domain.EmailCode{
		EmailAccountID: accounts[0].ID,
		UserID:         ctx.EffectiveUser.Id,
		Sender:         "noreply@testservice.com",
		Subject:        "Test verification code",
		Code:           testCode,
		RuleName:       "test",
		RawBodyHash:    fmt.Sprintf("test_%d", time.Now().UnixNano()),
		ReceivedAt:     time.Now(),
	})
	if err != nil {
		slog.Error("failed to create test code", "error", err)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка создания тестового кода.", nil)
		return nil
	}

	_, _ = b.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("🧪 <b>Тестовый код создан</b>\n\n<code>%s</code>  <b>TestService</b>\n\n<i>Нажмите 🔑 Коды, чтобы проверить отображение.</i>", testCode),
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return nil
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
		infos = append(infos, keyboards.EmailAccountInfo{ID: acc.ID, Email: acc.Email})
	}

	kb := keyboards.EmailAccountsKeyboard(infos)
	_, err = b.SendMessage(ctx.EffectiveChat.Id, "Выберите ящик для удаления:", &gotgbot.SendMessageOpts{ReplyMarkup: kb})
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
			sb.WriteString(fmt.Sprintf("   ⚠️ %s\n", *acc.LastError))
		}
	}

	_, err = b.SendMessage(ctx.EffectiveChat.Id, sb.String(), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// HandleCodes shows the last 10 intercepted codes (shared across all users).
func (h *Handler) HandleCodes(b *gotgbot.Bot, ctx *ext.Context) error {
	dbCtx := context.Background()

	codes, err := h.store.ListRecentCodes(dbCtx, 10)
	if err != nil {
		slog.Error("failed to list codes", "error", err)
		_, _ = b.SendMessage(ctx.EffectiveChat.Id, "❌ Ошибка. Попробуйте позже.", nil)
		return nil
	}
	if len(codes) == 0 {
		_, _ = b.SendMessage(ctx.EffectiveChat.Id,
			"🔑 <b>Кодов пока нет</b>\n\n"+
				"За последние 24 часа ничего не получено.\n\n"+
				"<i>Коды появятся здесь автоматически, как только придут на отслеживаемые почтовые ящики.</i>",
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return nil
	}

	msk := time.FixedZone("MSK", 3*60*60)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔑 <b>Последние коды</b> — %d\n\n", len(codes)))
	for _, code := range codes {
		age := time.Since(code.ReceivedAt)
		serviceName := extractServiceName(code.Sender)
		sb.WriteString(fmt.Sprintf("<code>%s</code>  <b>%s</b>\n", code.Code, serviceName))
		sb.WriteString(fmt.Sprintf("<i>%s · %s · %s</i>\n\n",
			code.Email,
			code.ReceivedAt.In(msk).Format("15:04"),
			formatAge(age)))
	}

	_, err = b.SendMessage(ctx.EffectiveChat.Id, sb.String(), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// extractServiceName converts sender email to a readable service name.
func extractServiceName(sender string) string {
	// Known mappings
	knownServices := map[string]string{
		"google":    "Google",
		"facebook":  "Facebook",
		"meta":      "Meta",
		"microsoft": "Microsoft",
		"apple":     "Apple",
		"amazon":    "Amazon",
		"yandex":    "Яндекс",
		"vk":        "ВКонтакте",
		"mail":      "Mail.ru",
		"telegram":  "Telegram",
		"instagram": "Instagram",
		"twitter":   "Twitter",
		"binance":   "Binance",
		"bybit":     "Bybit",
		"okx":       "OKX",
		"coinbase":  "Coinbase",
	}

	// Extract domain from "Name <email@domain>" or just "email@domain"
	s := strings.ToLower(sender)
	if idx := strings.LastIndex(s, "@"); idx >= 0 {
		domain := s[idx+1:]
		domain = strings.TrimSuffix(domain, ">")
		// Remove TLD
		parts := strings.Split(domain, ".")
		if len(parts) >= 2 {
			baseDomain := parts[len(parts)-2]
			if name, ok := knownServices[baseDomain]; ok {
				return name
			}
			// Return capitalized domain
			if len(baseDomain) > 0 {
				return strings.ToUpper(baseDomain[:1]) + baseDomain[1:]
			}
		}
		return domain
	}
	return sender
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "только что"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		return fmt.Sprintf("%d мин назад", m)
	}
	h := int(d.Hours())
	return fmt.Sprintf("%d ч назад", h)
}
