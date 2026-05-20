package core

import (
	"fmt"
	"strconv"

	"autofilterbot/internal/fsub"
	"autofilterbot/internal/functions"
	"autofilterbot/pkg/callbackdata"
	"slices"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// Select handles the select button callback query.
func Select(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "private" {
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

	ok, err = fsub.CheckFsub(_app, bot, ctx)
	if err != nil {
		if functions.IsChatNotFoundErr(err) { // user has not started bot or blocked
			// redirect to dm for a retry msg
			data := &RetryData{
				ChatId:    c.Message.GetChat().Id,
				MessageId: c.Message.GetMessageId(),
			}

			_, err = c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Url: fmt.Sprintf("t.me/%s?start=%s", bot.Username, data.Encode()),
			})
			if err != nil {
				_app.Log.Warn("all: retry answer failed", zap.Error(err))
			}

			return nil
		}

		_app.Log.Warn("all: check fsub failed", zap.Error(err))
	}

	if !ok {
		return nil
	}

	if len(data.Args) > 2 { // if file uid in args
		r.SelectFile(pageIndex, data.Args[2])
	}

	var (
		pageFiles = r.Files[pageIndex]
		buttons   = make([][]gotgbot.InlineKeyboardButton, 0, len(pageFiles)+2)
	)

	buttons = append(buttons, selectHeaderRow(r.UniqueId, pageIndex))
	
	menuButtons := pageFiles.SelectMenu(uniqueId, pageIndex)

	// Add Custom Button in the middle (after 5 rows or at the end)
	if _app.Config.ResultButtonText != "" && _app.Config.ResultButtonUrl != "" {
		splitPoint := 5
		if len(menuButtons) < splitPoint {
			splitPoint = len(menuButtons)
		}

		customBtn := []gotgbot.InlineKeyboardButton{{
			Text: _app.Config.ResultButtonText,
			Url:  _app.Config.ResultButtonUrl,
		}}

		menuButtons = slices.Insert(menuButtons, splitPoint, customBtn)
	}

	buttons = append(buttons, menuButtons...)
	buttons = append(buttons, selectFooterRow(r.UniqueId, pageIndex, len(r.Files)))

	_, _, err = c.Message.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
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
