package fsub

import (
	"fmt"
	"time"

	"autofilterbot/internal/config"
	"autofilterbot/internal/database/mongo"
	"autofilterbot/internal/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

var (
	membershipCache = NewFsubCache()
	antiSpamTTL     = 60 * time.Second
	antiSpamCache   = NewAntiSpamCache()
)

type appPreview interface {
	GetDB() *mongo.Client
	GetConfig() *config.Config
	GetLog() *zap.Logger
	BasicMessageValues(ctx *ext.Context, extraValues ...map[string]any) map[string]string
}

// GetMissingChannels returns channels the user hasn't joined.
// Uses 60s TTL cache to minimize API calls.
func GetMissingChannels(bot *gotgbot.Bot, db *mongo.Client, userId int64, channels []model.Channel) []model.Channel {
	var missing []model.Channel
	var userJoinReqs *model.User

	for _, ch := range channels {
		// 1. Check Cache
		if isMember, ok := membershipCache.Get(userId, ch.ID); ok && isMember {
			continue
		}

		// 2. Check getChatMember
		member, err := bot.GetChatMember(ch.ID, userId, nil)
		isMember := false
		if err == nil {
			status := member.GetStatus()
			if status == "creator" || status == "administrator" || status == "member" {
				isMember = true
			} else if status == "restricted" {
				if m, ok := member.(*gotgbot.ChatMemberRestricted); ok {
					isMember = m.IsMember
				} else if m, ok := member.(gotgbot.ChatMemberRestricted); ok {
					isMember = m.IsMember
				}
			}
		}

		if isMember {
			membershipCache.Set(userId, ch.ID, true, 60*time.Second)
			continue
		}

		// 3. Check Join Requests (Consider as Fsub if requested)
		if userJoinReqs == nil {
			userJoinReqs, _ = db.GetUserJoinRequests(userId)
		}
		hasReq := false
		if userJoinReqs != nil {
			for _, rid := range userJoinReqs.JoinRequests {
				if rid == ch.ID {
					hasReq = true
					break
				}
			}
		}

		if hasReq {
			membershipCache.Set(userId, ch.ID, true, 60*time.Second)
			continue
		}

		missing = append(missing, ch)
	}
	return missing
}

// CheckFsub is the main entry point for Fsub enforcement.
func CheckFsub(app appPreview, bot *gotgbot.Bot, ctx *ext.Context) (bool, error) {
	if ctx.EffectiveUser == nil || ctx.EffectiveUser.IsBot {
		return true, nil
	}

	userId := ctx.EffectiveUser.Id
	channels := app.GetConfig().GetFsubChannels()
	if len(channels) == 0 {
		return true, nil
	}

	missing := GetMissingChannels(bot, app.GetDB(), userId, channels)
	if len(missing) == 0 {
		return true, nil
	}

	// Sequential Fsub: Only show the FIRST channel from missing
	ch := missing[0]
	var btns [][]gotgbot.InlineKeyboardButton
	
	link := ch.InviteLink
	if link == "" {
		link = "https://t.me/telegram"
	}
	btns = append(btns, []gotgbot.InlineKeyboardButton{{Text: "ᴊᴏɪɴ " + ch.Title, Url: link}})
	btns = append(btns, []gotgbot.InlineKeyboardButton{{Text: "ᴛʀʏ ᴀɢᴀɪɴ 🔄", CallbackData: "fsub_verify"}})

	text := fmt.Sprintf("<b>📛 Aᴄᴄᴇss Dᴇɴɪᴇᴅ 📛</b>\n\n<i>𝖠𝖼𝖼𝖾𝗌𝗌 𝖨𝗌 𝖱𝖾𝗌𝗍𝗋𝗂𝖼𝗍𝖾𝖽. 𝖯𝗅𝖾𝖺𝗌𝖾 𝖲𝖾𝗇𝖽 𝖠 <b>𝖩𝗈𝗂𝗇 𝖱𝖾𝗊𝗎𝖾𝗌𝗍</b> 𝖳𝗈 𝖳𝗁𝖾 𝖢𝗁𝖺𝗇𝗇𝖾𝗅 𝖡𝖾𝗅𝗈𝗐 𝖳𝗈 𝖢𝗈𝗇𝗍𝗂𝗇𝗎𝖾.</i>\n\n<b>Channel [%d/%d]</b>", len(channels)-len(missing)+1, len(channels))

	// Private Chat Logic
	if ctx.EffectiveChat != nil && ctx.EffectiveChat.Type == "private" {
		// Store last action for resume
		action := ""
		if ctx.Message != nil {
			action = ctx.Message.Text
		} else if ctx.CallbackQuery != nil {
			action = "cb:" + ctx.CallbackQuery.Data
		}
		if action != "" {
			app.GetDB().SetUserLastAction(userId, action)
		}

		// Try to reuse existing message
		oldMsgId, _ := app.GetDB().GetUserFsubMessage(userId)
		if oldMsgId != 0 {
			_, _, err := bot.EditMessageText(text, &gotgbot.EditMessageTextOpts{
				ChatId:      userId,
				MessageId:   oldMsgId,
				ParseMode:   "HTML",
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
			})
			if err == nil {
				return false, nil
			}
		}

		// Fallback to new message
		msg, err := bot.SendMessage(userId, text, &gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
		})
		if err == nil {
			app.GetDB().SetUserFsubMessage(userId, msg.MessageId)
		}
		return false, err
	}

	// Non-private chat logic (group, supergroup, channel, or callback from outside private)
	if ctx.EffectiveChat != nil && ctx.EffectiveChat.Type != "private" {
		// Store last action so they can resume when they verify
		action := ""
		if ctx.CallbackQuery != nil {
			action = "cb:" + ctx.CallbackQuery.Data
		} else if ctx.Message != nil {
			action = ctx.Message.Text
		}
		if action != "" {
			app.GetDB().SetUserLastAction(userId, action)
		}

		// Attempt to DM the user
		msg, err := bot.SendMessage(userId, text, &gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
		})

		if err != nil {
			// DM failed (usually because they haven't started the bot in PM)
			if ctx.CallbackQuery != nil {
				// Show popup with redirect URL to PM
				_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "⚠️ Please start the bot first in PM and join the channel!",
					ShowAlert: true,
					Url:       fmt.Sprintf("https://t.me/%s?start=fsub", bot.Username),
				})
			} else if ctx.Message != nil {
				if antiSpamCache.ShouldWarn(userId, 5*time.Minute) {
					groupText := fmt.Sprintf("<b>Hey %s, Please Join Our Channels To Use The Bot!</b>\n\n<i>⚠️ I Couldn't DM You. Please Start The Bot First!</i>", ctx.EffectiveUser.FirstName)
					startBtn := gotgbot.InlineKeyboardButton{Text: "🚀 Sᴛᴀʀᴛ Bᴏᴛ", Url: "https://t.me/" + bot.Username + "?start=fsub"}
					_, _ = bot.SendMessage(ctx.EffectiveChat.Id, groupText, &gotgbot.SendMessageOpts{
						ParseMode: "HTML",
						ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{startBtn}}},
					})
				}
			}
		} else {
			// DM Succeeded!
			if msg != nil {
				app.GetDB().SetUserFsubMessage(userId, msg.MessageId)
			}
			if ctx.CallbackQuery != nil {
				_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "📩 I have sent the Join Requests / links to your PM! Please join/request and click this file button again.",
					ShowAlert: true,
				})
			} else if ctx.Message != nil {
				if antiSpamCache.ShouldWarn(userId, 5*time.Minute) {
					groupText := fmt.Sprintf("<b>Hey %s, Please Join Our Channels To Use The Bot!</b>\n\n<i>I've Sent The Join Links To Your DMs.</i>", ctx.EffectiveUser.FirstName)
					_, _ = bot.SendMessage(ctx.EffectiveChat.Id, groupText, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
				}
			}
		}
		return false, nil
	}

	return false, nil
}

// SetMembershipCache sets the cached membership status for a user and channel.
func SetMembershipCache(userId, channelId int64, isMember bool) {
	membershipCache.Set(userId, channelId, isMember, 60*time.Second)
}
