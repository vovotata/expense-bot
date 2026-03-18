package middleware

import (
	"log/slog"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// Logging logs incoming updates with structured fields.
func Logging(b *gotgbot.Bot, ctx *ext.Context) error {
	var (
		userID   int64
		username string
		msgType  string
	)

	if ctx.EffectiveUser != nil {
		userID = ctx.EffectiveUser.Id
		username = ctx.EffectiveUser.Username
	}

	switch {
	case ctx.Message != nil:
		if ctx.Message.Text != "" {
			msgType = "text"
		} else if len(ctx.Message.Photo) > 0 {
			msgType = "photo"
		} else if ctx.Message.Document != nil {
			msgType = "document"
		} else {
			msgType = "other_message"
		}
	case ctx.CallbackQuery != nil:
		msgType = "callback:" + ctx.CallbackQuery.Data
	default:
		msgType = "unknown"
	}

	slog.Info("update received",
		"user_id", userID,
		"username", username,
		"type", msgType,
	)

	return nil
}
