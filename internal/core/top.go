package core

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// TopSearching handles the top searching command and callback.
func TopSearching(bot *gotgbot.Bot, ctx *ext.Context) error {
	analytics := _app.Analytics.(*AnalyticsService)
	s, err := analytics.GetStats()
	if err != nil {
		_app.Log.Warn("top: get stats failed", zap.Error(err))
		return nil
	}

	var text strings.Builder
	text.WriteString("<b>🌟 Top 10 Trending Searches</b>\n\n")

	if len(s.TopSearches) == 0 {
		text.WriteString("<i>No trending searches yet. Start searching to see results here!</i>")
	} else {
		for i, st := range s.TopSearches {
			text.WriteString(fmt.Sprintf("%d. <code>%s</code> (%d searches)\n", i+1, st.Query, st.Count))
		}
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{
			Text: "« ʙᴀᴄᴋ", CallbackData: "cmd:start",
		}, {
			Text: "🗑️ ᴄʟᴏsᴇ", CallbackData: "close",
		}}},
	}

	if ctx.CallbackQuery != nil {
		_, _, err = ctx.EffectiveMessage.EditText(bot, text.String(), &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	} else {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, text.String(), &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	}

	return err
}
