package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"expense-bot/internal/config"
	"expense-bot/internal/emailwatch"
	"expense-bot/internal/storage/postgres"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg.LogLevel)

	store, err := postgres.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	bot, err := gotgbot.NewBot(cfg.BotToken, nil)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	notifier := emailwatch.NewTelegramNotifier(bot)
	watcher := emailwatch.NewWatcher(
		store, notifier, cfg.EmailEncryptionKey,
		cfg.EmailIDLETimeout, cfg.EmailReconnectMaxBackoff,
	)

	if err := watcher.Start(ctx); err != nil {
		slog.Error("failed to start watcher", "error", err)
		os.Exit(1)
	}

	slog.Info("email-watcher started")

	<-ctx.Done()
	slog.Info("shutting down email-watcher...")
	watcher.Stop()
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
