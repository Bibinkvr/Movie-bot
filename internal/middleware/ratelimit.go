package middleware

import (
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var (
	// userLastRequest tracks the last request time for each user.
	userLastRequest = make(map[int64][]time.Time)
	mu              sync.Mutex

	// Limit: 2 requests per 10 seconds
	limitDuration = 10 * time.Second
	limitCount    = 2
)

// RateLimit is a middleware that limits the number of requests a user can make.
func RateLimit(bot *gotgbot.Bot, ctx *ext.Context) error {
	var userID int64
	if ctx.EffectiveUser != nil {
		userID = ctx.EffectiveUser.Id
	} else {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	requests := userLastRequest[userID]

	// Filter out old requests
	var activeRequests []time.Time
	for _, t := range requests {
		if now.Sub(t) < limitDuration {
			activeRequests = append(activeRequests, t)
		}
	}

	if len(activeRequests) >= limitCount {
		// User is rate limited
		if ctx.CallbackQuery != nil {
			ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "𝖲𝗅𝗈𝗐 𝖣𝗈𝗐𝗇! 𝖯𝗅𝖾𝖺𝗌𝖾 𝗐𝖺𝗂𝗍 𝖺 𝖿𝖾𝗐 𝗌𝖾𝖼𝗈𝗇𝖽𝗌.",
				ShowAlert: true,
			})
		} else if ctx.Message != nil {
			ctx.Message.Reply(bot, "<b>⚠️ 𝖲𝗅𝗈𝗐 𝖣𝗈𝗐𝗇!</b>\n𝖸𝗈𝗎'𝗋𝖾 𝗌𝖾𝖺𝗋𝖼𝗁𝗂𝗇𝗀 𝗍𝗈𝗈 𝖿𝖺𝗌𝗍. 𝖯𝗅𝖾𝖺𝗌𝖾 𝗐𝖺𝗂𝗍 𝖺 𝖿𝖾𝗐 𝗌𝖾𝖼𝗈𝗇𝖽𝗌.", &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
		}
		return ext.EndGroups // Stop processing other handlers in the current group
	}

	activeRequests = append(activeRequests, now)
	userLastRequest[userID] = activeRequests

	return nil
}

// ClearRateLimitOldEntries periodically cleans up the map to prevent memory leaks.
func ClearRateLimitOldEntries() {
	for {
		time.Sleep(10 * time.Minute)
		mu.Lock()
		now := time.Now()
		for id, requests := range userLastRequest {
			var activeRequests []time.Time
			for _, t := range requests {
				if now.Sub(t) < limitDuration {
					activeRequests = append(activeRequests, t)
				}
			}
			if len(activeRequests) == 0 {
				delete(userLastRequest, id)
			} else {
				userLastRequest[id] = activeRequests
			}
		}
		mu.Unlock()
	}
}
