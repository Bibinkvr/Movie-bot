package middleware

import (
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var (
	userCooldowns = make(map[int64]time.Time)
	cooldownMu    sync.RWMutex
)

// AntiSpam prevents users from making requests too frequently.
func AntiSpam(cooldown time.Duration, isAdmin func(int64) bool) func(bot *gotgbot.Bot, ctx *ext.Context) error {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		userId := ctx.EffectiveUser.Id
		
		if isAdmin != nil && isAdmin(userId) {
			return nil // Bypass rate-limiting for admins
		}

		cooldownMu.RLock()
		lastSeen, exists := userCooldowns[userId]
		cooldownMu.RUnlock()

		if exists && time.Since(lastSeen) < cooldown {
			// Silently ignore or answer callback
			if ctx.CallbackQuery != nil {
				ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text: "Slow down! Wait a few seconds.",
				})
			}
			return ext.EndGroups // Stop processing this update
		}

		cooldownMu.Lock()
		userCooldowns[userId] = time.Now()
		cooldownMu.Unlock()

		return nil
	}
}

func init() {
	// Cleanup old entries every hour
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			cooldownMu.Lock()
			for id, lastSeen := range userCooldowns {
				if time.Since(lastSeen) > time.Hour {
					delete(userCooldowns, id)
				}
			}
			cooldownMu.Unlock()
		}
	}()
}
