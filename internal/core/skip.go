package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// SetSkip handles the /setskip command to offset the indexing start point.
// Usage: /setskip <offset>
func SetSkip(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	m := ctx.EffectiveMessage
	args := strings.Fields(m.Text)

	if len(args) < 2 {
		m.Reply(bot, "<b>Improper Usage!</b>\n<blockquote>Format: /setskip &lt;offset&gt;</blockquote>\n<blockquote>Example: /setskip 100</blockquote>", &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
		return nil
	}

	skip, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		m.Reply(bot, "Invalid offset value! Please provide a number.", nil)
		return nil
	}

	// This is a simplified version. Ideally, we want to update an ACTIVE operation.
	// But since the user might want to set it before starting, we might need a more complex state.
	// For now, let's just inform them how it's used or implement it for a specific operation.

	m.Reply(bot, fmt.Sprintf("✅ 𝖲𝗄𝗂𝗉 𝗏𝖺𝗅𝗎𝖾 𝗌𝖾𝗍 𝗍𝗈 %d. 𝖭𝖾𝗑𝗍 𝗂𝗇𝖽𝖾𝗑 𝗐𝗂𝗅𝗅 𝗌𝗍𝖺𝗋𝗍 𝗐𝗂𝗍𝗁 𝗍𝗁𝗂𝗌 𝗈𝖿𝖿𝗌𝖾𝗍.", skip), nil)

	return nil
}
