package core

import (
	"fmt"
	"strconv"
	"time"

	"autofilterbot/internal/autofilter"
	"autofilterbot/internal/fsub"
	"autofilterbot/internal/functions"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/callbackdata"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// Select handles the select button callback query.
func Select(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat == nil || ctx.EffectiveChat.Type != "private" {
		ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Please use this in Private Chat Only!", ShowAlert: true})
		return nil
	}
	c := ctx.CallbackQuery

	data := callbackdata.FromString(c.Data)
	if len(data.Args) < 2 {
		_app.Log.Warn("select: not enough args", zap.Strings("args", data.Args))

		_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Error: Not Enough Arguments", ShowAlert: true})

		return err
	}

	pageIndex, err := strconv.Atoi(data.Args[1])
	if err != nil {
		_app.Log.Warn("select: parse index failed", zap.Error(err))

		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Sorry An Error occurred :/", ShowAlert: true})

		return err
	}

	uniqueId := data.Args[0]

	r, ok, err := _app.Cache.Autofilter.Get(uniqueId)
	if !ok {
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Search Result Has Expired!\nPlease Try Again...", ShowAlert: true})
		return err
	}

	if err != nil {
		_app.Log.Warn("select: get result cache failed", zap.Error(err))

		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Sorry An Error occurred :/", ShowAlert: true})

		return err
	}

	if r.FromUser != c.From.Id {
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "You Can't Use This Button!", ShowAlert: true})
		return err
	}

	if pageIndex >= len(r.Files) {
		_app.Log.Warn("select: page not found", zap.Int("length", len(r.Files)), zap.Int("index", pageIndex))

		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Result Page Not Found!", ShowAlert: true})

		return err
	}

	if len(data.Args) > 2 { // if file uid in args
		r.SelectFile(pageIndex, data.Args[2])
	}

	var (
		pageFiles = r.Files[pageIndex]
		buttons   = make([][]gotgbot.InlineKeyboardButton, 0, len(pageFiles)+2)
	)

	buttons = append(buttons, selectHeaderRow(r.UniqueId, pageIndex))
	
	// Divider
	latestReleasesBtn := gotgbot.InlineKeyboardButton{Text: "🌟 Latest Releases 🌟", CallbackData: "ignore", Style: "success"}
	if _app.Config.LatestReleasesUrl != "" {
		latestReleasesBtn.Url = _app.Config.LatestReleasesUrl
		latestReleasesBtn.CallbackData = ""
	}
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{latestReleasesBtn})

	menuButtons := pageFiles.SelectMenu(uniqueId, pageIndex)
	buttons = append(buttons, menuButtons...)
	buttons = append(buttons, selectFooterRow(r.UniqueId, pageIndex, len(r.Files)))

	if c.InlineMessageId != "" {
		_, _, err = bot.EditMessageReplyMarkup(&gotgbot.EditMessageReplyMarkupOpts{
			InlineMessageId: c.InlineMessageId,
			ReplyMarkup:     gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		})
	} else if c.Message != nil {
		_, _, err = c.Message.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
			ChatId:      c.Message.GetChat().Id,
			MessageId:   c.Message.GetMessageId(),
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		})
	}
	if err != nil {
		_app.Log.Warn("select: edit markup failed", zap.Error(err))
	}

	err = _app.Cache.Autofilter.Save(r)
	if err != nil {
		_app.Log.Warn("select: save result failed", zap.Error(err))
	}

	return nil
}

func selectHeaderRow(uniqueId string, pageIndex int) []gotgbot.InlineKeyboardButton {
	return []gotgbot.InlineKeyboardButton{
		{Text: "ᴇxɪᴛ", CallbackData: fmt.Sprintf("navg|%s_%d", uniqueId, pageIndex), Style: "danger"},
		{Text: "sᴇɴᴅ ➡️", CallbackData: fmt.Sprintf("sendsel|%s", uniqueId), Style: "success"},
	}
}

func selectFooterRow(uniqueId string, pageIndex, totalPages int) []gotgbot.InlineKeyboardButton {
	btns := make([]gotgbot.InlineKeyboardButton, 0, 3)
	if pageIndex != 0 {
		btns = append(btns, selectBackButton(uniqueId, pageIndex-1))
	}

	btns = append(btns, gotgbot.InlineKeyboardButton{Text: fmt.Sprintf("📑 𝗣𝗔𝗚𝗘 %d/%d", pageIndex+1, totalPages), CallbackData: "ignore"})

	if pageIndex+1 != totalPages {
		btns = append(btns, selectNextButton(uniqueId, pageIndex+1))
	}

	return btns
}

func selectBackButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "« ʙᴀᴄᴋ", CallbackData: fmt.Sprintf("sel|%s_%d", uniqueId, pageIndex), Style: "primary"}
}

func selectNextButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "ɴᴇxᴛ »", CallbackData: fmt.Sprintf("sel|%s_%d", uniqueId, pageIndex), Style: "primary"}
}

// SendSelected handles the send selected button callback query.
func SendSelected(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat == nil || ctx.EffectiveChat.Type != "private" {
		ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Please use this in Private Chat Only!", ShowAlert: true})
		return nil
	}
	c := ctx.CallbackQuery

	data := callbackdata.FromString(c.Data)
	if len(data.Args) < 1 {
		_app.Log.Warn("sendsel: not enough args", zap.Strings("args", data.Args))
		_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Error: Not Enough Arguments", ShowAlert: true})
		return err
	}

	uniqueId := data.Args[0]

	r, ok, err := _app.Cache.Autofilter.Get(uniqueId)
	if !ok {
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Search Result Has Expired!\nPlease Try Again...", ShowAlert: true})
		return err
	}

	if err != nil {
		_app.Log.Warn("sendsel: get result cache failed", zap.Error(err))
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Sorry An Error occurred :/", ShowAlert: true})
		return err
	}

	if r.FromUser != c.From.Id {
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "You Can't Use This Button!", ShowAlert: true})
		return err
	}

	ok, err = fsub.CheckFsub(_app, bot, ctx)
	if !ok {
		return nil
	}

	// Gather all selected files
	var selectedFiles []autofilter.File
	for _, page := range r.Files {
		for _, f := range page {
			if f.IsSelected {
				selectedFiles = append(selectedFiles, f)
			}
		}
	}

	if len(selectedFiles) == 0 {
		_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "⚠️ No files selected! Please select some files first.",
			ShowAlert: true,
		})
		return err
	}

	sentMessages := make([]struct {
		chatId    int64
		messageId int64
	}, 0, len(selectedFiles))

	var (
		warn    string
		delTime = _app.Config.GetFileAutoDelete()
	)
	if delTime != 0 {
		warn = fmt.Sprintf("<blockquote><b><i>⚠️ 𝖳𝗁𝗂𝗌 𝖥𝗂𝗅𝖾 𝖶𝗂𝗅𝗅 𝖻𝖾 𝖠𝗎𝗍𝗈𝗆𝖺𝗍𝗂𝖼𝖺𝗅lh 𝖣𝖾𝗅𝖾𝗍𝖾𝖽 𝗂𝗇 %d 𝖬𝗂𝗇𝗎𝗍𝖾𝗌. 𝖥𝗈𝗋𝗐𝖺𝗋𝖽 𝗂𝗍 𝗍𝗈 𝖠𝗇𝗈𝗍𝗁𝖾𝗋 𝖢𝗁𝖺𝗍 𝗈𝗋 𝖲𝖺𝗏𝖾𝖽 𝖬𝖾𝗌𝗌𝖺𝗀𝖾𝗌.</i></b></blockquote>", delTime)
	}

	for _, f := range selectedFiles {
		msg, err := f.Send(bot, c.From.Id, &model.SendFileOpts{
			Caption: _app.FormatText(ctx, _app.Config.GetFileCaption(), map[string]any{
				"file_size": functions.FileSizeToString(f.FileSize),
				"file_name": autofilter.CleanFileNameForDisplay(f.FileName),
				"warn":      warn,
			}),
			Keyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "🗑️ ᴅᴇʟᴇᴛᴇ ғɪʟᴇ 🗑️", CallbackData: "close"}}},
		})
		if err != nil {
			if functions.IsChatNotFoundErr(err) { // user has not started bot or blocked
				var msgChatId, msgId int64
				if c.Message != nil {
					msgChatId = c.Message.GetChat().Id
					msgId = c.Message.GetMessageId()
				}
				retryData := &RetryData{
					ChatId:    msgChatId,
					MessageId: msgId,
				}

				_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Url: fmt.Sprintf("https://t.me/%s?start=%s", bot.Username, retryData.Encode()),
				})
				if err != nil {
					_app.Log.Warn("sendsel: retry answer failed", zap.Error(err))
				}

				return nil
			}

			_app.Log.Warn("sendsel: send file failed", zap.Error(err), zap.String("file_id", f.FileId))
			continue
		}

		sentMessages = append(sentMessages, struct {
			chatId    int64
			messageId int64
		}{chatId: msg.Chat.Id, messageId: msg.MessageId})
	}

	_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      fmt.Sprintf("%d selected files have been sent privately 🥳", len(sentMessages)),
		ShowAlert: true,
	})
	if err != nil {
		_app.Log.Warn("sendsel: answer query failed", zap.Error(err))
	}

	if delTime != 0 {
		duration := time.Minute * time.Duration(delTime)
		for _, m := range sentMessages {
			err = _app.AutoDelete.SaveData(m.chatId, m.messageId, duration)
			if err != nil {
				_app.Log.Warn("sendsel: save autodelete failed", zap.Error(err))
			}
		}
	}

	return nil
}
