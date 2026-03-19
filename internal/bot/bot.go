package bot

import (
	"context"
	"fmt"
	"log/slog"

	apphandlers "expense-bot/internal/bot/handlers"
	"expense-bot/internal/bot/middleware"
	"expense-bot/internal/config"
	"expense-bot/internal/fsm"
	"expense-bot/internal/storage"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

type Bot struct {
	api        *gotgbot.Bot
	updater    *ext.Updater
	dispatcher *ext.Dispatcher
	handler    *apphandlers.Handler
	cfg        *config.Config
}

func New(ctx context.Context, cfg *config.Config, store storage.Storage, fsmStore fsm.StateStore) (*Bot, error) {
	api, err := gotgbot.NewBot(cfg.BotToken, nil)
	if err != nil {
		return nil, fmt.Errorf("bot.New: %w", err)
	}

	slog.Info("bot authorized", "username", api.User.Username)

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			slog.Error("dispatcher error", "error", err)
			return ext.DispatcherActionNoop
		},
	})

	updater := ext.NewUpdater(dispatcher, nil)

	h := apphandlers.New(ctx, store, fsmStore, cfg.AdminChatID, cfg.AdminUserIDs, cfg.EmailEncryptionKey)

	b := &Bot{
		api:        api,
		updater:    updater,
		dispatcher: dispatcher,
		handler:    h,
		cfg:        cfg,
	}

	b.registerMiddleware()
	b.registerHandlers()
	return b, nil
}

func (b *Bot) registerMiddleware() {
	// Group -2: Logging (always runs)
	b.dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, middleware.Logging), -2)
	b.dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.All, middleware.Logging), -2)

	// Group -1: Whitelist + Rate limiting
	wl := middleware.NewWhitelist(b.cfg.AllowedUserIDs)
	b.dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, wl.Middleware), -1)
	b.dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.All, wl.Middleware), -1)

	rl := middleware.NewRateLimiter(b.cfg.RateLimitPerMin)
	b.dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, rl.Middleware), -1)
	b.dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.All, rl.Middleware), -1)
}

func (b *Bot) registerHandlers() {
	// Group 0: Command handlers
	b.dispatcher.AddHandler(handlers.NewCommand("start", b.handler.SendMainMenu))
	b.dispatcher.AddHandler(handlers.NewCommand("menu", b.handler.SendMainMenu))
	b.dispatcher.AddHandler(handlers.NewCommand("addmail", b.handler.HandleAddMail))
	b.dispatcher.AddHandler(handlers.NewCommand("delmail", b.handler.HandleDelMail))
	b.dispatcher.AddHandler(handlers.NewCommand("mymails", b.handler.HandleMyMails))
	b.dispatcher.AddHandler(handlers.NewCommand("codes", b.handler.HandleCodes))
	b.dispatcher.AddHandler(handlers.NewCommand("testcode", b.handler.HandleTestCode))

	// Callback queries
	b.dispatcher.AddHandler(handlers.NewCallback(callbackquery.All, b.handler.HandleCallback))

	// Photo messages (for QR/screenshots)
	b.dispatcher.AddHandler(handlers.NewMessage(message.Photo, b.handler.HandlePhoto))
	b.dispatcher.AddHandler(handlers.NewMessage(message.Document, b.handler.HandlePhoto))

	// Text messages (wizard input) — must be last
	b.dispatcher.AddHandler(handlers.NewMessage(message.Text, b.handler.HandleTextMessage))
}

func (b *Bot) Start() error {
	return b.updater.StartPolling(b.api, &ext.PollingOpts{
		DropPendingUpdates: true,
	})
}

func (b *Bot) Stop() {
	b.updater.Stop()
}
