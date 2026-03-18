package bot

import (
	"fmt"
	"log/slog"

	apphandlers "expense-bot/internal/bot/handlers"
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
}

func New(cfg *config.Config, store storage.Storage, fsmStore fsm.StateStore) (*Bot, error) {
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

	h := apphandlers.New(store, fsmStore, cfg.AdminChatID)

	b := &Bot{
		api:        api,
		updater:    updater,
		dispatcher: dispatcher,
		handler:    h,
	}

	b.registerHandlers()
	return b, nil
}

func (b *Bot) registerHandlers() {
	b.dispatcher.AddHandler(handlers.NewCommand("start", b.handler.Start))
	b.dispatcher.AddHandler(handlers.NewCommand("addmail", b.handler.HandleAddMail))
	b.dispatcher.AddHandler(handlers.NewCommand("delmail", b.handler.HandleDelMail))
	b.dispatcher.AddHandler(handlers.NewCommand("mymails", b.handler.HandleMyMails))
	b.dispatcher.AddHandler(handlers.NewCommand("codes", b.handler.HandleCodes))

	// Callback queries
	b.dispatcher.AddHandler(handlers.NewCallback(callbackquery.All, b.handler.HandleCallback))

	// Photo messages (for QR/screenshots)
	b.dispatcher.AddHandler(handlers.NewMessage(message.Photo, b.handler.HandlePhoto))
	b.dispatcher.AddHandler(handlers.NewMessage(message.Document, b.handler.HandlePhoto))

	// Text messages (wizard input)
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
