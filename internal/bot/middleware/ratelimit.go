package middleware

import (
	"sync"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"golang.org/x/time/rate"
)

// RateLimiter limits requests per user.
type RateLimiter struct {
	limiters sync.Map
	perMin   int
}

// NewRateLimiter creates a rate limiter with the given limit per minute.
func NewRateLimiter(perMin int) *RateLimiter {
	return &RateLimiter{perMin: perMin}
}

func (rl *RateLimiter) getLimiter(userID int64) *rate.Limiter {
	val, ok := rl.limiters.Load(userID)
	if ok {
		return val.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rate.Limit(float64(rl.perMin)/60.0), rl.perMin)
	actual, _ := rl.limiters.LoadOrStore(userID, limiter)
	return actual.(*rate.Limiter)
}

// Middleware returns an ext handler function that enforces rate limiting.
func (rl *RateLimiter) Middleware(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveUser == nil {
		return nil
	}

	limiter := rl.getLimiter(ctx.EffectiveUser.Id)
	if !limiter.Allow() {
		if ctx.CallbackQuery != nil {
			_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "Слишком много запросов. Подождите немного.",
				ShowAlert: true,
			})
		} else if ctx.EffectiveMessage != nil {
			_, _ = ctx.EffectiveMessage.Reply(b, "⚠️ Слишком много запросов. Подождите немного.", nil)
		}
		// Return ext.EndGroups equivalent — we stop processing by returning a special error
		return ext.EndGroups
	}

	return nil
}
