package core

import (
	"fmt"
	"strconv"
	"strings"

	"autofilterbot/internal/config"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/conversation"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// AddToFsub handles the fsub_add:<id> callback.
func AddToFsub(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	data := ctx.CallbackQuery.Data
	split := strings.Split(data, ":")
	if len(split) < 2 {
		return nil
	}

	chatID, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return err
	}

	chat, err := bot.GetChat(chatID, nil)
	if err != nil {
		ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Failed to get chat info!", ShowAlert: true})
		return err
	}

	currentChannels := _app.Config.GetFsubChannels()
	for _, c := range currentChannels {
		if c.ID == chat.Id {
			ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Already in Fsub!", ShowAlert: true})
			return nil
		}
	}

	// Create invite link (requires bot to be admin)
	link, err := bot.CreateChatInviteLink(chat.Id, &gotgbot.CreateChatInviteLinkOpts{
		Name:               "ForceSub",
		CreatesJoinRequest: true, // Default to true for better "as given method" experience
	})
	if err != nil {
		_app.Log.Warn("fsub_add: failed to create invite link", zap.Error(err))
		ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Failed to create invite link. Make sure bot is Admin!", ShowAlert: true})
		return nil
	}

	currentChannels = append(currentChannels, model.Channel{
		ID:                 chat.Id,
		Title:              chat.Title,
		InviteLink:         link.InviteLink,
		CreatesJoinRequest: true,
	})

	err = _app.DB.UpdateConfig(bot.Id, config.FieldNameFsub, currentChannels)
	if err != nil {
		_app.Log.Error("fsub_add: failed to update config", zap.Error(err))
		return err
	}

	go _app.RefreshConfig()

	ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Added to Fsub Successfully! ✅"})
	ctx.CallbackQuery.Message.EditText(bot, fmt.Sprintf("✅ <b>%s</b> has been added to Force Subscribe channels.", chat.Title), &gotgbot.EditMessageTextOpts{ParseMode: gotgbot.ParseModeHTML})

	return nil
}

// PromptIndexCallback handles the index_prompt:<id> callback.
func PromptIndexCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	data := ctx.CallbackQuery.Data
	split := strings.Split(data, ":")
	if len(split) < 2 {
		return nil
	}

	chatID, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return err
	}

	// For indexing, we need a message ID. We'll use the last message ID of the channel.
	chat, err := bot.GetChat(chatID, nil)
	if err != nil {
		return err
	}

	// We can't easily get the last message ID without a message from the channel.
	// But we can prompt the user to forward a message or tell them we'll start from 1.
	
	// Actually, let's just trigger the same prompt as AutoDetectIndex but with the ID fixed.
	return PromptIndex(bot, ctx, chat.Id, 0) // messageID 0 means we'll ask or assume 1
}
// SetFsub handles the /fsub command to set multiple channels.
func SetFsub(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	var ids []string
	var replyToMsg *gotgbot.Message

	if ctx.CallbackQuery != nil {
		conv := conversation.NewConversatorFromUpdate(bot, ctx.Update)
		askM, err := conv.Ask(_app.Ctx, "<b>𝖯𝗅𝖾𝖺𝗌𝖾 𝗌𝖾𝗇𝖽 𝗍𝗁𝖾 𝖥𝗈𝗋𝖼𝖾𝖲𝗎𝖻 𝖼𝗁𝖺𝗇𝗇𝖾𝗅 𝖨𝖣𝗌 (separated by spaces or commas):</b>", &gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "❌ Cancel", CallbackData: "admin:cancel"}}},
			},
			ParseMode: gotgbot.ParseModeHTML,
		})
		if err != nil {
			return nil
		}
		rawInput := strings.TrimSpace(askM.Text)
		rawInput = strings.ReplaceAll(rawInput, ",", " ")
		ids = strings.Fields(rawInput)
		replyToMsg = askM
	} else {
		args := ctx.Args()
		if len(args) < 2 {
			_, err := ctx.Message.Reply(bot, "<b>𝖴𝗌𝖺𝗀𝖾:</b> <code>/fsub &lt;ID1&gt; &lt;ID2&gt; ...</code>\n<i>𝖸𝗈𝗎 𝖼𝖺𝗇 𝗎𝗌𝖾 𝗌𝗉𝖺𝖼𝖾𝗌 𝗈𝗋 𝖼𝗈𝗆𝗆𝖺𝗌 𝗍𝗈 𝗌𝖾𝗉𝖺𝗋𝖺𝗍𝖾 𝖨𝖣𝗌.</i>", &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
			return err
		}
		rawArgs := strings.Join(args[1:], " ")
		rawArgs = strings.ReplaceAll(rawArgs, ",", " ")
		ids = strings.Fields(rawArgs)
		replyToMsg = ctx.Message
	}

	var (
		newChannels []model.Channel
		success     []string
		failed      []string
	)

	for _, idStr := range ids {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (Invalid ID)", idStr))
			continue
		}

		chat, err := bot.GetChat(id, nil)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%d (Not Found/No Access)", id))
			continue
		}

		link, err := bot.CreateChatInviteLink(chat.Id, &gotgbot.CreateChatInviteLinkOpts{
			Name:               "ForceSub",
			CreatesJoinRequest: true,
		})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%d (No Admin Privileges)", id))
			continue
		}

		newChannels = append(newChannels, model.Channel{
			ID:                 chat.Id,
			Title:              chat.Title,
			InviteLink:         link.InviteLink,
			CreatesJoinRequest: true,
		})
		success = append(success, chat.Title)
	}

	if len(newChannels) > 0 {
		err := _app.DB.UpdateConfig(bot.Id, config.FieldNameFsub, newChannels)
		if err != nil {
			_app.Log.Error("set_fsub: update config failed", zap.Error(err))
			if replyToMsg != nil {
				_, err = replyToMsg.Reply(bot, "<b>📛 Database Update Failed!</b>", &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
			}
			return err
		}
		_app.RefreshConfig()
	}

	res := "<b>Force Subscribe Update Result:</b>\n\n"
	if len(success) > 0 {
		res += "✅ <b>Success:</b> " + strings.Join(success, ", ") + "\n"
	}
	if len(failed) > 0 {
		res += "❌ <b>Failed:</b> " + strings.Join(failed, ", ") + "\n"
	}

	if replyToMsg != nil {
		_, err := replyToMsg.Reply(bot, res, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
		return err
	}
	return nil
}
