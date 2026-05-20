package core

import (
	"strconv"

	"autofilterbot/internal/fsub"
	"autofilterbot/internal/autofilter"
	"autofilterbot/pkg/callbackdata"
	"slices"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// Navigate handles the navg callback query from autofilter results for pagination.
func Navigate(bot *gotgbot.Bot, ctx *ext.Context) error {
	isPrivate := ctx.EffectiveChat.Type == "private"
	if isPrivate {
		ok, _ := fsub.CheckFsub(_app, bot, ctx)
		if !ok {
			return nil
		}
	}
	c := ctx.CallbackQuery

	data := callbackdata.FromString(c.Data)
	if len(data.Args) < 2 {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Error: Not Enough Arguments", ShowAlert: true, CacheTime: fiveHoursInSeconds})
		return nil
	}

	r, ok, err := _app.Cache.Autofilter.Get(data.Args[0])
	if err != nil {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "An Error occurred :\\", ShowAlert: true})
		_app.Log.Warn("navg: result from cache failed", zap.Error(err))
		return nil
	}

	if !ok {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "This Query Has Expired!\nPlease Request Again...", ShowAlert: true, CacheTime: fiveHoursInSeconds})
		return nil
	}

	if r.FromUser != c.From.Id {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "You Can't Use This, Please Ask For Your Own!", ShowAlert: true, CacheTime: fiveHoursInSeconds})
		return nil
	}

	pageIndex, err := strconv.Atoi(data.Args[1])
	if err != nil {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "An Error occurred :\\", ShowAlert: true})
		_app.Log.Warn("navg: parse page index failed", zap.Error(err))
		return nil
	}

	files := r.Files

	if pageIndex > len(files)-1 {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "404: Result Page Not Found", ShowAlert: true})
		_app.Log.Warn("navg: result page not found", zap.String("unique_id", r.UniqueId), zap.Int("index", pageIndex))
		return nil
	}

	pageFiles := files[pageIndex]

	var (
		buttons   = make([][]gotgbot.InlineKeyboardButton, 0, len(pageFiles)+2)
		chatId    = c.Message.GetChat().Id
	)
	if isPrivate {
		isSeries := autofilter.DetectType(pageFiles) == "series"
		buttons = append(buttons, headerRow(r.UniqueId, pageIndex, isSeries))
	}

	fileButtons := pageFiles.Process(chatId, bot.Username, _app.Config)

	// Add Custom Button in the middle (after 5 rows or at the end)
	if _app.Config.ResultButtonText != "" && _app.Config.ResultButtonUrl != "" {
		splitPoint := 5
		if len(fileButtons) < splitPoint {
			splitPoint = len(fileButtons)
		}

		customBtn := []gotgbot.InlineKeyboardButton{{
			Text: _app.Config.ResultButtonText,
			Url:  _app.Config.ResultButtonUrl,
		}}

		fileButtons = slices.Insert(fileButtons, splitPoint, customBtn)
	}

	buttons = append(buttons, fileButtons...)
	buttons = append(buttons, footerRow(r.UniqueId, pageIndex, len(files)))

	_, _, err = c.Message.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ChatId:    chatId,
		MessageId: c.Message.GetMessageId(),
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
	if err != nil {
		_app.Log.Warn("navg: edit markup failed", zap.Error(err), zap.String("unique_id", r.UniqueId))
	}

	return nil
}
