// Handles batch and gelink cmds. Lots of scruffy spaghetti code.

package core

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"autofilterbot/internal/fsub"
	"autofilterbot/internal/functions"
	"autofilterbot/pkg/conversation"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// SendBatch sends a message batch to the target chat.
func SendBatch(bot *gotgbot.Bot, chatID int64, rawData string) error {
	data, err := BatchURLDataFromString(rawData)
	if err != nil {
		return err
	}

	for i := data.StartMessageId; i <= data.EndMessageId; i++ {
		_, err := bot.CopyMessage(chatID, data.ChatId, i, nil)
		if err != nil {
			if functions.IsChatNotFoundErr(err) {
				bot.SendMessage(chatID, "It looks like I'm no longer a part of the channel where this batch was created :(\n\nPlease contact a bot administrator or add me to the channel if you are one.", nil)
				_app.Log.Debug("sendbatch: failed to send batch: chat not found", zap.Int64("channel_id", data.ChatId), zap.Error(err))

				return nil
			}

			_app.Log.Debug("sendbatch: copy message failed", zap.Int64("channel_id", data.ChatId), zap.Int64("msg_id", i), zap.Error(err))
		}
	}

	return nil
}

// NewBatch handles the /batch commmand.
func NewBatch(bot *gotgbot.Bot, ctx *ext.Context) error {
	ok, _ := fsub.CheckFsub(_app, bot, ctx)
	if !ok {
		return nil
	}
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	m := ctx.EffectiveMessage
	chatId := m.Chat.Id

	var (
		channelId, startId, endId int64
	)

	if replyM := m.ReplyToMessage; replyM != nil {
		if origin, ok := replyM.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
			channelId = origin.Chat.Id
			startId = origin.MessageId
		} else if link, err := functions.ParseMessageLink(replyM.Text); err == nil {
			if c, err := link.GetChat(bot); err == nil {
				channelId = c.Id
				startId = link.MessageId
			} else {
				sendChatErr(bot, chatId, err)
				return nil
			}
		}
	}

	if ctx.CallbackQuery == nil {
		split := strings.Fields(m.Text)
		if len(split) > 1 {
			if link, err := functions.ParseMessageLink(split[1]); err == nil {
				if startId != 0 {
					endId = link.MessageId
				} else {
					if c, err := link.GetChat(bot); err == nil {
						channelId = c.Id
						startId = link.MessageId
					} else {
						sendChatErr(bot, chatId, err)
						return nil
					}
				}
			}

			if len(split) > 2 && endId == 0 {
				if link, err := functions.ParseMessageLink(split[2]); err == nil {
					if startId != 0 {
						endId = link.MessageId
					} else {
						if c, err := link.GetChat(bot); err == nil {
							channelId = c.Id
							startId = link.MessageId
						} else {
							sendChatErr(bot, chatId, err)
							return nil
						}
					}
				}
			}
		}
	}

	if startId == 0 {
		conv := conversation.NewConversatorFromUpdate(bot, ctx.Update)

		askM, err := conv.Ask(_app.Ctx, "Please forward or send the post link of the first message in the batch:", &gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "❌ Cancel", CallbackData: "admin:cancel"}}},
			},
		})
		if err != nil {
			_app.Log.Debug("batch: conv exited with error", zap.Error(err))
			return nil
		}

		if origin, ok := askM.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
			channelId = origin.Chat.Id
			startId = origin.MessageId
		} else if link, err := functions.ParseMessageLink(askM.Text); err == nil {
			if c, err := link.GetChat(bot); err == nil {
				channelId = c.Id
				startId = link.MessageId
			} else {
				sendChatErr(bot, chatId, err)
				return nil
			}
		} else {
			askM.Reply(bot, "Message Is Not a Forwarded Channel Post or Message Link!", nil)
			return nil
		}
	}

	if endId == 0 {
		conv := conversation.NewConversatorFromUpdate(bot, ctx.Update)

		askM, err := conv.Ask(_app.Ctx, "Please forward or send the post link of the last message in the batch:", &gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "❌ Cancel", CallbackData: "admin:cancel"}}},
			},
		})
		if err != nil {
			_app.Log.Debug("batch: conv exited with error", zap.Error(err))
			return nil
		}

		if origin, ok := askM.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
			endId = origin.MessageId
		} else if link, err := functions.ParseMessageLink(askM.Text); err == nil {
			endId = link.MessageId
		} else {
			askM.Reply(bot, "Message Is Not a Forwarded Channel Post or Message Link!", nil)
			return nil
		}
	}

	if startId > endId {
		m.Reply(bot, "First Message Cannot be After The Last :/", nil)
		return nil
	}

	if endId-startId > _app.Config.GetBatchSizeLimit() {
		m.Reply(bot, "Batch Too Large :/\n\nCreate a Smaller Batch or Update The Batch Size Limit From the Config Panel!", nil)
		_app.Log.Debug("batch: too large", zap.Int64("chat_id", channelId), zap.Int64("start", startId), zap.Int64("end", endId))
		return nil
	}

	data := &BatchURLData{
		ChatId:         channelId,
		StartMessageId: startId,
		EndMessageId:   endId,
	}
	url := fmt.Sprintf("https://t.me/%s?start=%s", bot.Username, data.Encode())

	text := fmt.Sprintf(`
<b>𝖬𝖾𝗌𝗌𝖺𝗀𝖾 𝖡𝖺𝗍𝖼𝗁 𝖧𝖺𝗌 𝖡𝖾𝖾𝗇 𝖢𝗋𝖾𝖺𝗍𝖾𝖽 𝖲𝗎𝖼𝖼𝖾𝗌𝗌𝖿𝗎𝗅𝗅𝗒 🎉</b>
<b>𝖳𝗋𝗒 𝖭𝗈𝗐:</b> <a href='%s'>ᴄʟɪᴄᴋ ʜᴇʀᴇ</a>
<b>𝖢𝗈𝗉𝗒:</b> <code>%s</code>
`, url, url)
	btn := [][]gotgbot.InlineKeyboardButton{
		{{Text: "𝖳𝗋𝗒 𝖭𝗈𝗐", Url: url}},
		{{Text: "𝖳𝖺𝗉 𝗍𝗈 𝖢𝗈𝗉𝗒", CopyText: &gotgbot.CopyTextButton{Text: url}}},
	}

	_, err := bot.SendMessage(m.Chat.Id, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btn},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	if err != nil {
		_app.Log.Warn("batch: send success msg failed", zap.Error(err))
	}

	return nil
}

// GenLink handles the /genlink command. Mostly copies from batch.
func GenLink(bot *gotgbot.Bot, ctx *ext.Context) error {
	ok, _ := fsub.CheckFsub(_app, bot, ctx)
	if !ok {
		return nil
	}
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	m := ctx.EffectiveMessage
	chatId := m.Chat.Id

	var (
		channelId, messageId int64
	)

	if replyM := m.ReplyToMessage; replyM != nil {
		if origin, ok := replyM.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
			channelId = origin.Chat.Id
			messageId = origin.MessageId
		} else if link, err := functions.ParseMessageLink(replyM.Text); err == nil {
			if c, err := link.GetChat(bot); err == nil {
				channelId = c.Id
				messageId = link.MessageId
			} else {
				sendChatErr(bot, chatId, err)
				return nil
			}
		}
	}

	if messageId == 0 && ctx.CallbackQuery == nil {
		split := strings.Fields(m.Text)
		if len(split) > 1 {
			if link, err := functions.ParseMessageLink(split[1]); err == nil {
				if c, err := link.GetChat(bot); err == nil {
					channelId = c.Id
					messageId = link.MessageId
				} else {
					sendChatErr(bot, chatId, err)
					return nil
				}
			}
		}
	}

	if messageId == 0 {
		if replyM := m.ReplyToMessage; replyM != nil {
			var msgId *gotgbot.MessageId
			var err error
			if replyM.Document != nil || replyM.Video != nil || replyM.Audio != nil {
				// Forward to log channel to get a permanent link
				msgId, err = bot.CopyMessage(_app.LogChannel, chatId, replyM.MessageId, nil)
				if err != nil {
					_app.Log.Error("genlink: forward to log failed", zap.Error(err), zap.Int64("log_channel", _app.LogChannel))
					m.Reply(bot, "❌ Failed to process file. Ensure LOG_CHANNEL is correct and Bot is admin!", nil)
					return nil
				}
				channelId = _app.LogChannel
				messageId = msgId.MessageId
			}
		}
	}

	if messageId == 0 {
		conv := conversation.NewConversatorFromUpdate(bot, ctx.Update)

		askM, err := conv.Ask(_app.Ctx, "Please forward or send the post link of the message:", &gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "❌ Cancel", CallbackData: "admin:cancel"}}},
			},
		})
		if err != nil {
			_app.Log.Debug("genlink: conv exited with error", zap.Error(err))
			return nil
		}

		if origin, ok := askM.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
			channelId = origin.Chat.Id
			messageId = origin.MessageId
		} else if link, err := functions.ParseMessageLink(askM.Text); err == nil {
			if c, err := link.GetChat(bot); err == nil {
				channelId = c.Id
				messageId = link.MessageId
			} else {
				sendChatErr(bot, chatId, err)
				return nil
			}
		} else {
			askM.Reply(bot, "Message Is Not a Forwarded Channel Post or Message Link!", nil)
			return nil
		}
	}

	data := &BatchURLData{
		ChatId:         channelId,
		StartMessageId: messageId,
		EndMessageId:   messageId,
	}
	url := fmt.Sprintf("https://t.me/%s?start=%s", bot.Username, data.Encode())

	text := fmt.Sprintf(`
<b>𝖬𝖾𝗌𝗌𝖺𝗀𝖾 𝖫𝗂𝗇𝗄 𝖧𝖺𝗌 𝖡𝖾𝖾𝗇 𝖢𝗋𝖾𝖺𝗍𝖾𝖽 𝖲𝗎𝖼𝖼𝖾𝗌𝗌𝖿𝗎𝗅𝗅𝗒 🎉</b>
<b>𝖳𝗋𝗒 𝖭𝗈𝗐:</b> <a href='%s'>ᴄʟɪᴄᴋ ʜᴇʀᴇ</a>
<b>𝖢𝗈𝗉𝗒:</b> <code>%s</code>
`, url, url)
	btn := [][]gotgbot.InlineKeyboardButton{
		{{Text: "𝖳𝗋𝗒 𝖭𝗈𝗐", Url: url}},
		{{Text: "𝖳𝖺𝗉 𝗍𝗈 𝖢𝗈𝗉𝗒", CopyText: &gotgbot.CopyTextButton{Text: url}}},
	}

	_, err := bot.SendMessage(m.Chat.Id, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btn},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	if err != nil {
		_app.Log.Warn("genlink: send success msg failed", zap.Error(err))
	}

	return nil
}

// BatchURLData is the url data from the start command.
type BatchURLData struct {
	ChatId         int64
	StartMessageId int64
	EndMessageId   int64
}

func (d *BatchURLData) Encode() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("b_%d_%d_%d", d.ChatId, d.StartMessageId, d.EndMessageId)))
}

// BatchURLDataFromString converts a string, usually the start data, to batch url data.
func BatchURLDataFromString(s string) (*BatchURLData, error) {
	split := strings.Split(s, "_")
	if len(split) < 4 {
		return nil, errors.New("not enough arguments")
	}

	chatId, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return nil, err
	}

	startId, err := strconv.ParseInt(split[2], 10, 64)
	if err != nil {
		return nil, err
	}

	endId, err := strconv.ParseInt(split[3], 10, 64)
	if err != nil {
		return nil, err
	}

	return &BatchURLData{
		ChatId:         chatId,
		StartMessageId: startId,
		EndMessageId:   endId,
	}, nil
}

// sendChatError sends an error message alerting the user the given chat was not found.
func sendChatErr(bot *gotgbot.Bot, chatId int64, err error) {
	bot.SendMessage(chatId, "📛 𝖳𝗁𝗂𝗌 𝖢𝗁𝖺𝗍 𝖶𝖺𝗌 𝖭𝗈𝗍 𝖥𝗈𝗎𝗇𝖽!\n𝖤𝗇𝗌𝗎𝗋𝖾 𝗍𝗁𝖾 𝖡𝗈𝗍 𝗂𝗌 𝖺𝗇 𝖠𝖽𝗆𝗂𝗇 𝖳𝗁𝖾𝗋𝖾 𝖺𝗇𝖽 𝖥𝗈𝗋𝗐𝖺𝗋𝖽𝖾𝖽 𝖬𝖾𝗌𝗌𝖺𝗀𝖾 𝗈𝗋 𝖫𝗂𝗇𝗄 𝖶𝖺𝗌 𝖢𝗈𝗋𝗋𝖾𝖼𝗍 :/", nil)
	_app.Log.Debug("getChat failed", zap.Error(err))
}
