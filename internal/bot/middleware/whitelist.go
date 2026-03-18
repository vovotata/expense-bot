package middleware

import (
	"log/slog"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// Whitelist restricts access to allowed user IDs.
type Whitelist struct {
	allowed map[int64]struct{}
}

// NewWhitelist creates a whitelist middleware. Empty slice = allow all.
func NewWhitelist(userIDs []int64) *Whitelist {
	allowed := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		allowed[id] = struct{}{}
	}
	return &Whitelist{allowed: allowed}
}

// Middleware checks if the user is allowed.
func (w *Whitelist) Middleware(b *gotgbot.Bot, ctx *ext.Context) error {
	if len(w.allowed) == 0 {
		return nil // all users allowed
	}

	if ctx.EffectiveUser == nil {
		return nil
	}

	if _, ok := w.allowed[ctx.EffectiveUser.Id]; !ok {
		slog.Warn("unauthorized access attempt",
			"user_id", ctx.EffectiveUser.Id,
			"username", ctx.EffectiveUser.Username,
		)
		if ctx.EffectiveMessage != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "⛔ У вас нет доступа к этому боту.", nil)
		}
		return ext.EndGroups
	}

	return nil
}
