package middleware

import (
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"golang.org/x/time/rate"
)

type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter limits requests per user with automatic cleanup of idle entries.
type RateLimiter struct {
	limiters sync.Map
	perMin   int
	done     chan struct{}
}

// NewRateLimiter creates a rate limiter with periodic cleanup of idle entries.
func NewRateLimiter(perMin int) *RateLimiter {
	rl := &RateLimiter{perMin: perMin, done: make(chan struct{})}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Stop() {
	close(rl.done)
}

func (rl *RateLimiter) getLimiter(userID int64) *rate.Limiter {
	now := time.Now()
	val, ok := rl.limiters.Load(userID)
	if ok {
		ul := val.(*userLimiter)
		ul.lastSeen = now
		return ul.limiter
	}
	ul := &userLimiter{
		limiter:  rate.NewLimiter(rate.Limit(float64(rl.perMin)/60.0), rl.perMin),
		lastSeen: now,
	}
	actual, _ := rl.limiters.LoadOrStore(userID, ul)
	return actual.(*userLimiter).limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.limiters.Range(func(key, value any) bool {
				ul := value.(*userLimiter)
				if time.Since(ul.lastSeen) > time.Hour {
					rl.limiters.Delete(key)
				}
				return true
			})
		case <-rl.done:
			return
		}
	}
}

// Middleware enforces rate limiting per user.
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
		return ext.EndGroups
	}

	return nil
}
