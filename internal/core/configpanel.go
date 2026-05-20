package core

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// Settings handles the /settings command which acts as the entrypoint into the config panel.
func Settings(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	m := ctx.Message

	_, err := m.Reply(bot, "<b>⚙️ Cʟɪᴄᴋ Tʜᴇ Bᴜᴛᴛᴏɴ Bᴇʟᴏᴡ Tᴏ Oᴘᴇɴ Tʜᴇ Cᴏɴғɪɢ Pᴀɴᴇʟ 👇</b>", &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "𝖮𝖯𝖤𝖭", CallbackData: "config"}},
				{{Text: "🔙 Back", CallbackData: "admin:back"}},
			},
		},
		ParseMode: gotgbot.ParseModeHTML,
	})
	if err != nil {
		_app.Log.Warn("send settings msg failed", zap.Error(err))
	}

	return nil
}

// ConfigPanel handles callback queries for the config panel.
func ConfigPanel(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	err := _app.ConfigPanel.HandleUpdate(ctx, bot)
	if err != nil {
		_app.Log.Warn("handle config panel failed", zap.Error(err))
	}

	return nil
}
