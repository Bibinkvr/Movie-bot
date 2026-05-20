package core

import (
	"fmt"
	"autofilterbot/internal/fsub"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// HandleJoinRequest handles join requests to force subsribe channels.
func HandleJoinRequest(bot *gotgbot.Bot, ctx *ext.Context) error {
	update := ctx.ChatJoinRequest

	// saves all join requests, not just from fsub channels
	err := _app.DB.SaveUserJoinRequest(update.From.Id, update.Chat.Id)
	if err != nil {
		_app.Log.Warn("handlejoinrequest: failed to save join request to db", zap.Int64("user_id", update.From.Id), zap.Int64("chat_id", update.Chat.Id), zap.Error(err))
	}

	// Sequential Fsub Transition Logic
	go func() {
		userId := update.From.Id
		msgId, _ := _app.DB.GetUserFsubMessage(userId)
		if msgId == 0 {
			return // No active fsub prompt
		}

		channels := _app.Config.GetFsubChannels()
		missing := fsub.GetMissingChannels(bot, _app.GetDB(), userId, channels)

		if len(missing) > 0 {
			// Still missing some channels, update the prompt to the next one
			ch := missing[0]
			link := ch.InviteLink
			if link == "" {
				link = "https://t.me/telegram"
			}
			text := fmt.Sprintf("<b>✅ 𝖱𝖾𝗊𝗎𝖾𝗌𝗍 𝖱𝖾𝖼𝖾𝗂𝗏𝖾𝖽 𝖿𝗈𝗋 𝖢𝗁𝖺𝗇𝗇𝖾𝗅 %d!</b>\n\n<i>𝖭𝗈𝗐, 𝖯𝗅𝖾𝖺𝗌𝖾 𝖲𝖾𝗇𝖽 𝖠 𝖩𝗈𝗂𝗇 𝖱𝖾𝗊𝗎𝖾𝗌𝗍 𝖳𝗈 𝖳𝗁𝖾 𝖭𝖾𝗑𝗍 𝖢𝗁𝖺𝗇𝗇𝖾𝗅 𝖡𝖾𝗅𝗈𝗐.</i>\n\n<b>Channel [%d/%d]</b>", len(channels)-len(missing), len(channels)-len(missing)+1, len(channels))
			btns := [][]gotgbot.InlineKeyboardButton{
				{{Text: "ᴊᴏɪɴ " + ch.Title, Url: link}},
				{{Text: "✨ I Rᴇǫᴜᴇsᴛᴇᴅ ✨", CallbackData: "fsub_verify"}},
			}

			_, _, err := bot.EditMessageText(text, &gotgbot.EditMessageTextOpts{
				ChatId:      userId,
				MessageId:   msgId,
				ParseMode:   "HTML",
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
			})
			if err != nil {
				_app.Log.Debug("handlejoinrequest: failed to update fsub message", zap.Error(err))
			}
		} else {
			// All channels requested! Auto-verify and send success
			_app.Log.Info("handlejoinrequest: user finished sequential fsub", zap.Int64("user_id", userId))
			
			// Delete the prompt
			_, _ = bot.DeleteMessage(userId, msgId, nil)
			_app.DB.SetUserFsubMessage(userId, 0)

			// Trigger success logic instantly
			_ = ResumeUserAction(bot, ctx, userId)
		}
	}()

	return nil
}

// HandleChatMember handles chat member updates to cleanup join requests.
func HandleChatMember(bot *gotgbot.Bot, ctx *ext.Context) error {
	update := ctx.ChatMember
	newStatus := update.NewChatMember.GetStatus()

	// If they became a member, remove from JoinRequests collection
	if newStatus == "member" || newStatus == "administrator" || newStatus == "creator" {
		err := _app.DB.DeleteUserJoinRequest(update.NewChatMember.GetUser().Id, update.Chat.Id)
		if err == nil {
			_app.Log.Debug("handlechatmember: removed user from join requests (joined)", zap.Int64("user_id", update.NewChatMember.GetUser().Id), zap.Int64("chat_id", update.Chat.Id))
		}
	}

	return nil
}
