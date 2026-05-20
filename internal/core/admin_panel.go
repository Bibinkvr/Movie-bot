package core

import (
	"strings"

	"autofilterbot/pkg/conversation"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// AdminPanel sends the admin dashboard with inline buttons.
func AdminPanel(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	text := "<b>🛠️ ADMIN PANEL</b>\n\n<i>Select a category below to manage the bot:</i>"
	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			// Row 1: Overview
			{{Text: "Stats", CallbackData: "admin:stats"}, {Text: "Users", CallbackData: "admin:users"}},
			// Row 2: Messaging
			{{Text: "Broadcast", CallbackData: "admin:broadcast"}},
			// Row 3: File Management
			{{Text: "Index", CallbackData: "admin:index"}, {Text: "Batch", CallbackData: "admin:batch"}, {Text: "GenLink", CallbackData: "admin:genlink"}},
			// Row 4: Delete Controls
			{{Text: "Delete", CallbackData: "admin:delete"}, {Text: "Delete All", CallbackData: "admin:deleteall"}, {Text: "Clean Quality", CallbackData: "admin:clean"}},
			// Row 5: Configuration
			{{Text: "Settings", CallbackData: "admin:settings"}},
			// Row 6: Navigation
			{{Text: "Close", CallbackData: "admin:close"}},
		},
	}

	var err error
	if ctx.CallbackQuery != nil {
		_, _, err = ctx.CallbackQuery.Message.EditText(bot, text, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	} else {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	}

	if err != nil {
		_app.Log.Warn("admin_panel: send/edit failed", zap.Error(err))
	}

	return nil
}

// AdminCallbackHandler handles all callbacks starting with "admin:".
func AdminCallbackHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	c := ctx.CallbackQuery
	data := c.Data
	action := strings.TrimPrefix(data, "admin:")

	// Answer callback to remove loading state
	c.Answer(bot, nil)

	switch action {
	case "back":
		return AdminPanel(bot, ctx)
	case "cancel":
		conversation.Cancel(ctx.EffectiveChat.Id, c.From.Id)
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Operation Cancelled ❌"})
		_, err := c.Message.Delete(bot, nil)
		return err
	case "stats", "users":
		return Stats(bot, ctx)
	case "broadcast":
		return Broadcast(bot, ctx)
	case "index":
		return CmdIndex(bot, ctx)
	case "batch":
		return NewBatch(bot, ctx)
	case "genlink":
		return GenLink(bot, ctx)
	case "delete":
		return DeleteFile(bot, ctx)
	case "deleteall":
		return DeleteAllFiles(bot, ctx)
	case "clean":
		return CleanQuality(bot, ctx)
	case "settings":
		return Settings(bot, ctx)
	case "close":
		return Close(bot, ctx)
	default:
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Unknown Admin Action", ShowAlert: true})
	}

	return nil
}
