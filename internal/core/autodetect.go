package core

import (
	"fmt"
	"regexp"
	"strconv"

	"autofilterbot/internal/functions"
	"autofilterbot/internal/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

var (
	linkRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/(?:c/)?([a-zA-Z0-9_]{5,}|[0-9]{5,})/([0-9]+)`)
	idRegex   = regexp.MustCompile(`-100[0-9]{10,}`)
)

// AutoDetectIndex listens for forwarded messages or links to prompt for indexing.
func AutoDetectIndex(bot *gotgbot.Bot, ctx *ext.Context) error {
	defer func() {
		if r := recover(); r != nil {
			_app.Log.Error("[autodetect][AutoDetectIndex] panic recovered", zap.Any("panic", r))
		}
	}()

	// Check admin silently
	isAdmin := false
	for _, id := range _app.Admins {
		if ctx.EffectiveUser != nil && id == ctx.EffectiveUser.Id {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return nil
	}

	m := ctx.EffectiveMessage
	text := m.GetText()

	// Ignore commands
	if text != "" && text[0] == '/' {
		return nil
	}

	// 1. Check for forwarded channel message
	if origin, ok := m.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
		return PromptIndex(bot, ctx, origin.Chat.Id, origin.MessageId)
	}

	// 2. Check for message link in text using regex
	if match := linkRegex.FindString(text); match != "" {
		link, err := functions.ParseMessageLink(match)
		if err == nil {
			return PromptIndex(bot, ctx, link.ChatId, link.MessageId)
		}
		_app.Log.Debug("[autodetect] parse link failed", zap.String("match", match), zap.Error(err))
	}

	// 3. Scan for raw Chat IDs
	foundIDs := idRegex.FindAllString(text, -1)
	if len(foundIDs) > 0 {
		processed := make(map[int64]bool)
		for _, s := range foundIDs {
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil || processed[id] {
				continue
			}
			processed[id] = true

			chat, err := bot.GetChat(id, nil)
			if err != nil {
				_app.Log.Warn("[autodetect] getchat failed", zap.Int64("id", id), zap.Error(err))
				continue
			}

			msgText := fmt.Sprintf("<b>🔍 𝖢𝗁𝖺𝗇𝗇𝖾𝗅 𝖣𝖾𝗍𝖾𝖼𝗍𝖾𝖽:</b> <code>%s</code>\n<b>🆔 𝖨𝖣:</b> <code>%d</code>\n\n<b>𝖶𝗁𝖺𝗍 𝖽𝗈 𝗒𝗈𝗎 𝗐𝖺𝗇𝗍 𝗍𝗈 𝖽𝗈?</b>", chat.Title, chat.Id)
			btns := [][]gotgbot.InlineKeyboardButton{{
				{Text: "➕ 𝖠𝖽𝖽 𝗉𝗈 𝖥𝗌𝗎𝖻", CallbackData: fmt.Sprintf("fsub_add:%d", chat.Id)},
				{Text: "📂 𝖨𝗇𝖽𝖾𝗑 𝖢𝗁𝖺𝗇𝗇𝖾𝗅", CallbackData: fmt.Sprintf("index_prompt:%d", chat.Id)},
			}}

			_, err = m.Reply(bot, msgText, &gotgbot.SendMessageOpts{
				ParseMode:   gotgbot.ParseModeHTML,
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns},
			})
			if err != nil {
				_app.Log.Error("[autodetect] failed to reply", zap.Error(err))
			}
		}
		return nil
	}

	return nil
}

func PromptIndex(bot *gotgbot.Bot, ctx *ext.Context, channelID, messageID int64) error {
	m := ctx.EffectiveMessage
	
	// Check if already an operation exists to avoid duplicates
	// (Optional improvement, but let's keep it simple for now)

	plainChannelID := channelID
	if channelID < -1000000000000 {
		plainChannelID = (channelID + 1000000000000) * -1
	}

	text := fmt.Sprintf("<b>📂 𝖣𝗈 𝖸𝗈𝗎 𝖶𝖺𝗇𝗍 𝖳𝗈 𝖨𝗇𝖽𝖾𝗑 𝖳𝗁𝗂𝗌 𝖢𝗁𝖺𝗇𝗇𝖾𝗅?</b>\n\n<b>🆔 𝖢𝗁𝖺𝗍 𝖨𝖣:</b> <code>%d</code>\n<b>📍 𝖫𝖺𝗌𝗍 𝖬𝖾𝗌𝗌𝖺𝗀𝖾:</b> <a href='https://t.me/c/%d/%d'>%d</a>", channelID, plainChannelID, messageID, messageID)

	indexModel := model.Index{
		ID:                    functions.RandString(6),
		StartMessageID:        1,
		EndMessageID:          messageID,
		CurrentMessageID:      1,
		ChannelID:             channelID,
		ProgressMessageChatID: ctx.EffectiveChat.Id,
		ProgressMessageID:     0, // Will be updated if we send a new message, but for now we'll send a reply
		IsPaused:              true,
	}

	err := _app.DB.NewIndexOperation(&indexModel)
	if err != nil {
		_app.Log.Error("[autodetect] failed to create index operation", zap.Error(err))
		return nil
	}

	keyboard := [][]gotgbot.InlineKeyboardButton{{
		indexModel.CancelButton(), 
		indexModel.ModifyButton(), 
		indexModel.StartButton(),
	}}

	sentM, err := m.Reply(bot, text, &gotgbot.SendMessageOpts{
		ParseMode:   gotgbot.ParseModeHTML,
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err == nil && sentM != nil {
		_app.DB.UpdateIndexOperation(indexModel.ID, map[string]any{"pmessage_id": sentM.MessageId})
	}

	return err
}
