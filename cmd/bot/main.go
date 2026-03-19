package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"expense-bot/internal/bot"
	"expense-bot/internal/config"
	"expense-bot/internal/fsm"
	"expense-bot/internal/storage/postgres"
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

	fsmStore := fsm.NewMemoryStore(cfg.FSMTTL)
	defer fsmStore.Stop()

	b, err := bot.New(ctx, cfg, store, fsmStore)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := b.Start(); err != nil {
		slog.Error("failed to start bot", "error", err)
		os.Exit(1)
	}

	slog.Info("bot started, waiting for updates...")

	<-ctx.Done()
	slog.Info("shutting down...")
	b.Stop()
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
