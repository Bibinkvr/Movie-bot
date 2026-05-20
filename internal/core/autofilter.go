package core

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"autofilterbot/internal/autofilter"
	"autofilterbot/internal/button"
	"autofilterbot/internal/format"
	"autofilterbot/internal/fsub"
	"autofilterbot/internal/functions"
	"autofilterbot/internal/middleware"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/callbackdata"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

func Autofilter(bot *gotgbot.Bot, ctx *ext.Context) error {
	// 1. FSub check
	isPrivate := ctx.EffectiveChat.Type == "private"
	if isPrivate {
		ok, err := fsub.CheckFsub(_app, bot, ctx)
		if err != nil {
			_app.Log.Warn("autofilter: check fsub failed", zap.Error(err))
		}
		if !ok {
			return nil
		}
	}

	// 2. Wrap search in a job and push to queue
	job := Job{
		Bot: bot,
		Ctx: ctx,
		Handler: func(bot *gotgbot.Bot, ctx *ext.Context) error {
			msg, err := _autofilter(bot, ctx)
			if err != nil {
				return err
			}
			if msg != nil && _app.Config.GetAutodeleteTime() != 0 {
				_app.AutoDelete.SaveMessage(msg, time.Minute*time.Duration(_app.Config.AutodeleteTime))
			}
			return nil
		},
	}

	if pushed := _app.WorkerPool.(*WorkerPool).Push(job); !pushed {
		// Queue full, maybe notify user or just ignore
		_app.Log.Warn("autofilter: job queue full, skipping search")
	}

	// Register user message for auto-deletion
	if ctx.Message != nil {
		delTime := _app.Config.GetAutodeleteTime()
		if delTime != 0 {
			err := _app.AutoDelete.SaveMessage(ctx.Message, time.Minute*time.Duration(delTime))
			if err != nil {
				_app.Log.Warn("autofilter: save user query autodelete failed", zap.Error(err))
			}
		}
	}

	return nil
}

// autofilter runs the autofilter task and returns the sent message.
func _autofilter(bot *gotgbot.Bot, ctx *ext.Context) (*gotgbot.Message, error) {
	var (
		query        string
		inputMessage gotgbot.MaybeInaccessibleMessage
		fromUser     *gotgbot.User
	)

	switch {
	case ctx.CallbackQuery != nil:
		c := ctx.CallbackQuery

		callbackData := callbackdata.FromString(c.Data)
		if callbackData.Path[0] == "suggest" || callbackData.Path[0] == "reset" {
			query = callbackData.Args[0]
			inputMessage = c.Message
			fromUser = &c.From
		} else {
			if len(callbackData.Args) < 2 {
				_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "Malformed Query: Not Enough Arguments",
					ShowAlert: true,
				})
				_app.Log.Warn("autofilter: bad callback data", zap.Strings("args", callbackData.Args))
				return nil, err
			}

			userId, err := strconv.ParseInt(callbackData.Args[1], 10, 64)
			if err != nil {
				_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "Sorry An Error occurred :{",
					ShowAlert: true,
				})
				_app.Log.Warn("autofilter: parse user id failed", zap.Error(err))
				return nil, err
			}

			if c.From.Id != userId {
				_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
					Text:      "You Can't Use This Button!",
					ShowAlert: true,
				})
				return nil, err
			}

			inputMessage = c.Message
			if m, ok := c.Message.(*gotgbot.Message); ok {
				inputMessage = m.ReplyToMessage
			}

			query = callbackData.Args[0]
			fromUser = &c.From
		}
	case ctx.Message != nil:
		m := ctx.Message

		text := m.Text
		if text == "" {
			return nil, nil
		}

		if autofilter.IsBadQuery(text, m.Entities) {
			_app.Log.Debug("autofilter: bad query", zap.String("text", text), zap.Any("entities", m.Entities))
			return nil, nil
		}

		text = autofilter.Sanitize(text)

		inputMessage = m
		query = text
		fromUser = m.From
	default:
		_app.Log.Warn("autofilter: unsupported update type", zap.Int64("update_id", ctx.UpdateId))
		return nil, nil
	}

	// 3. Track search language stats
	go func() {
		lang := functions.DetectLanguage(query)
		if lang != "" {
			_app.DB.IncrementGlobalLangStat(_app.Config.BotId, lang)
			_app.DB.IncrementUserLangStat(fromUser.Id, lang)
		}
	}()

	cursor, err := _app.DB.SearchFiles(query)
	if err != nil {
		_app.Log.Warn("autofilter: search files failed", zap.Error(err))
		return bot.SendMessage(inputMessage.GetChat().Id, "<i>I'm Having Some Database Issues Right Now 😓\nPlease Try Again Later!</i>", &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	files, err := autofilter.FilesFromCursor(context.Background(), cursor, _app.Config)
	if err != nil {
		_app.Log.Warn("autofilter: files from cursor failed", zap.Error(err))
		return bot.SendMessage(inputMessage.GetChat().Id, "<i>Processing Results Failed 🤖</i>", &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ParseMode: gotgbot.ParseModeHTML,
		})
	}

	if len(files) == 0 {
		// Try a more relaxed search for spelling suggestions
		var suggestion string
		relaxedQuery := strings.ReplaceAll(query, " ", "")
		if len(relaxedQuery) > 3 {
			// Try a search with wildcards between characters
			var wildcardQuery string
			for _, r := range relaxedQuery {
				wildcardQuery += string(r) + ".*"
			}
			cursor, err := _app.DB.SearchFiles(wildcardQuery)
			if err == nil {
				f, _ := autofilter.FilesFromCursor(context.Background(), cursor, _app.Config)
				if len(f) > 0 && len(f[0]) > 0 {
					suggestion = f[0][0].FileName
				}
			}
		}

		vals := _app.BasicMessageValues(ctx, map[string]any{"query": query})
		text := format.KeyValueFormat(_app.Config.GetNoResultText(), vals)

		buttons := [][]gotgbot.InlineKeyboardButton{
			{{Text: "Sᴇᴀʀᴄʜ Oɴ Gᴏᴏɢʟᴇ 🔎", Url: fmt.Sprintf("https://google.com/?q=%s", query), Style: "primary"}},
			{{Text: "Cᴏᴘʏ", CopyText: &gotgbot.CopyTextButton{Text: query}, Style: "primary"}, button.Close(fromUser.Id)},
		}

		if suggestion != "" {
			text += fmt.Sprintf("\n\n<i>💡 D𝗂𝖽 𝗒𝗈𝗎 𝗆𝖾𝖺𝗇:</i> <b>%s</b> ?", suggestion)

			// Sanitize and truncate suggestion for callback data to avoid BUTTON_DATA_INVALID (64 byte limit)
			callbackQuery := autofilter.Sanitize(suggestion)
			if len(callbackQuery) > 40 {
				callbackQuery = callbackQuery[:40]
			}

			suggestBtn := gotgbot.InlineKeyboardButton{
				Text:         fmt.Sprintf("🔎 Search: %s", suggestion),
				CallbackData: fmt.Sprintf("suggest|%s_%d", callbackQuery, fromUser.Id),
				Style:        "primary",
			}
			buttons = append([][]gotgbot.InlineKeyboardButton{{suggestBtn}}, buttons...)
		}

		return bot.SendMessage(inputMessage.GetChat().Id, text, &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: inputMessage.GetMessageId(),
			},
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
			ParseMode:   gotgbot.ParseModeHTML,
		})
	}

	var warn string
	if _app.Config.GetAutodeleteTime() != 0 {
		warn = fmt.Sprintf("<blockquote>○ 𝖠𝗎𝗍𝗈-𝖣𝖾𝗅𝖾𝗍𝖾: <b>%d 𝗆𝗂𝗇𝗌</b></blockquote>", _app.Config.AutodeleteTime)
	}

	var (
		buttons   = make([][]gotgbot.InlineKeyboardButton, 0, 10)
		uniqueId  = functions.RandString(15)
		isPrivate = inputMessage.GetChat().Type == "private"

		// Combine all results for detection
		allFiles []autofilter.File
	)
	for _, page := range files {
		allFiles = append(allFiles, page...)
	}

	searchType := autofilter.DetectType(allFiles)
	isSeries := searchType == "series"

	if isSeries {
		// Season row
		seasonButtons := autofilter.Files(allFiles).ProcessSeasons(uniqueId)
		// Set first season button to success style (Green) as per image
		if len(seasonButtons) > 0 && len(seasonButtons[0]) > 0 {
			seasonButtons[0][0].Style = "success"
		}
		buttons = append(buttons, seasonButtons...)

		// Language row
		languages := autofilter.DetectLanguages(allFiles)
		if len(languages) > 0 {
			var langRow []gotgbot.InlineKeyboardButton
			for _, l := range languages {
				langRow = append(langRow, gotgbot.InlineKeyboardButton{
					Text:         l,
					CallbackData: fmt.Sprintf("lang|%s_%s", uniqueId, l),
					Style:        "primary",
				})
				if len(langRow) == 2 {
					buttons = append(buttons, langRow)
					langRow = nil
				}
			}
			if len(langRow) > 0 {
				buttons = append(buttons, langRow)
			}
		}
	}

	fileButtons := files[0].Process(inputMessage.GetChat().Id, bot.Username, _app.Config)

	// Divider
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{{Text: "🌟 Latest Releases 🌟", CallbackData: "ignore", Style: "success"}})

	// Add file buttons
	buttons = append(buttons, fileButtons...)

	// Multi-select
	if isPrivate {
		buttons = append(buttons, []gotgbot.InlineKeyboardButton{{Text: "✅ Select Multiple Files", CallbackData: fmt.Sprintf("sel|%s_0", uniqueId), Style: "primary"}})
	}

	// Footer Action Row 1
	newMoviesBtn := gotgbot.InlineKeyboardButton{Text: "🍿 New Movies", Style: "success", CallbackData: "ignore"}
	if _app.Config.NewMoviesUrl != "" {
		newMoviesBtn.Url = _app.Config.NewMoviesUrl
		newMoviesBtn.CallbackData = ""
	}
	updatesBtn := gotgbot.InlineKeyboardButton{Text: "📺 Updates", Style: "success", CallbackData: "ignore"}
	if _app.Config.UpdatesUrl != "" {
		updatesBtn.Url = _app.Config.UpdatesUrl
		updatesBtn.CallbackData = ""
	}

	buttons = append(buttons, []gotgbot.InlineKeyboardButton{newMoviesBtn, updatesBtn})

	// Navigation
	buttons = append(buttons, footerRow(uniqueId, 0, len(files)))

	// Footer Action Row 2
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{
		{Text: "📢 Share", SwitchInlineQuery: &query},
		{Text: "❌ Close", CallbackData: "close", Style: "danger"},
		{Text: "♻️ Reset", CallbackData: "reset|" + query, Style: "primary"},
	})

	text := format.KeyValueFormat(_app.Config.GetResultTemplate(), _app.BasicMessageValues(ctx, map[string]any{
		"query":         query,
		"warn":          warn,
		"total":         len(allFiles),
		"results_count": len(allFiles),
	}))

	msg, err := bot.SendMessage(inputMessage.GetChat().Id, text, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			MessageId: inputMessage.GetMessageId(),
		},
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	if err != nil {
		_app.Log.Warn("autofilter: send result failed", zap.Error(err))
	} else {
		middleware.ReactWithRandomEmoji(bot, msg.Chat.Id, msg.MessageId, _app.Config, _app.Log)
	}

	err = _app.Cache.Autofilter.Save(&autofilter.SearchResult{
		UniqueId: uniqueId,
		Query:    query,
		FromUser: fromUser.Id,
		ChatID:   ctx.EffectiveChat.Id,
		Files:    files,
	})
	if err != nil {
		_app.Log.Warn("autfilter: save cache failed", zap.Error(err), zap.String("query", query))
	}

	return msg, nil
}

// InlineSearch handles telegram inline queries.
func InlineSearch(bot *gotgbot.Bot, ctx *ext.Context) error {
	iq := ctx.InlineQuery
	query := iq.Query

	if query == "" {
		// Optional: Show trending or help
		_, err := iq.Answer(bot, nil, &gotgbot.AnswerInlineQueryOpts{
			CacheTime:  func(v int64) *int64 { return &v }(300),
			IsPersonal: true,
			Button: &gotgbot.InlineQueryResultsButton{
				Text:           "Type movie name to search...",
				StartParameter: "inline_help",
			},
		})
		return err
	}

	cursor, err := _app.DB.SearchFiles(query)
	if err != nil {
		_app.Log.Warn("inline: search files failed", zap.Error(err), zap.String("query", query))
		return nil
	}

	files, err := autofilter.FilesFromCursor(context.Background(), cursor, _app.Config)
	if err != nil {
		_app.Log.Warn("inline: files from cursor failed", zap.Error(err))
		return nil
	}

	allFiles := slices.Concat(files...)

	if len(allFiles) == 0 {
		_, err := iq.Answer(bot, nil, &gotgbot.AnswerInlineQueryOpts{
			CacheTime: func(v int64) *int64 { return &v }(60),
			Button: &gotgbot.InlineQueryResultsButton{
				Text:           "No results found!",
				StartParameter: "no_results",
			},
		})
		return err
	}

	// Limit to 50 results as per Telegram API
	if len(allFiles) > 50 {
		allFiles = allFiles[:50]
	}

	results := make([]gotgbot.InlineQueryResult, 0, len(allFiles))
	for _, f := range allFiles {
		// Encode start parameter for the file
		data := autofilter.URLData{
			FileUniqueId: f.UniqueId,
			ChatId:       0, // Not applicable for inline
			HasShortener: false,
		}
		encoded := data.Encode()

		description := fmt.Sprintf("Size: %s", functions.FileSizeToString(f.FileSize))

		results = append(results, gotgbot.InlineQueryResultArticle{
			Id:    f.UniqueId,
			Title: f.FileName,
			InputMessageContent: gotgbot.InputTextMessageContent{
				MessageText: fmt.Sprintf("<b>File Found: %s</b>\n\nClick below to get the file.", f.FileName),
				ParseMode:   gotgbot.ParseModeHTML,
			},
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{{Text: "🎁 Get File", Url: fmt.Sprintf("https://t.me/%s?start=%s", bot.Username, encoded), Style: "success"}},
				},
			},
			Description: description,
		})
	}

	_, err = iq.Answer(bot, results, &gotgbot.AnswerInlineQueryOpts{
		CacheTime:  func(v int64) *int64 { return &v }(300),
		IsPersonal: true,
	})
	if err != nil {
		_app.Log.Warn("inline: answer failed", zap.Error(err))
	}

	return err
}

// SendFileCallback handles the direct file selection callback from private chats.
func SendFileCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery
	_, fileUniqueId, found := strings.Cut(c.Data, "|")
	if !found {
		return nil
	}
	f, err := _app.DB.GetFile(fileUniqueId)
	if err != nil {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "404: File Not Found!", ShowAlert: true})
		return nil
	}

	c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Sending File... 📤"})

	var (
		warn    string
		delTime = _app.Config.GetFileAutoDelete()
	)
	if delTime != 0 {
		warn = fmt.Sprintf("<blockquote><b>⚠️ This File Will Be Automatically Deleted in %d Minutes.\n\nPlease Forward it to Another Chat or Saved Messages to save it forever! 📥</b></blockquote>", delTime)
	}

	msg, err := f.Send(bot, c.Message.GetChat().Id, &model.SendFileOpts{
		Caption: _app.FormatText(ctx, _app.Config.GetFileCaption(), map[string]any{
			"file_size": functions.FileSizeToString(f.FileSize),
			"file_name": f.FileName,
			"warn":      warn,
		}),
		Keyboard: [][]gotgbot.InlineKeyboardButton{{{Text: "🗑️ ᴅᴇʟᴇᴛᴇ ғɪʟᴇ 🗑️", CallbackData: "close"}}},
	})
	if err != nil {
		_app.Log.Warn("callback: send file failed", zap.Error(err), zap.String("file_id", f.FileId))
		return nil
	}

	if delTime != 0 && msg != nil {
		err = _app.AutoDelete.SaveMessage(msg, time.Minute*time.Duration(delTime))
		if err != nil {
			_app.Log.Warn("callback: insert auto delete failed", zap.Error(err))
		}
	}

	return nil
}

func headerRow(uniqueId string, pageIndex int, isSeries bool) []gotgbot.InlineKeyboardButton {
	row := []gotgbot.InlineKeyboardButton{selectButton(uniqueId, pageIndex)}
	if isSeries {
		row = append(row, gotgbot.InlineKeyboardButton{Text: "📂 sᴇᴀsᴏɴs", CallbackData: "af|" + uniqueId, Style: "primary"})
	}
	return row
}

func allButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "ᴀʟʟ", CallbackData: fmt.Sprintf("all|%s_%d", uniqueId, pageIndex), Style: "success"}
}

func selectButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "sᴇʟᴇᴄᴛ", CallbackData: fmt.Sprintf("sel|%s_%d", uniqueId, pageIndex), Style: "primary"}
}

func footerRow(uniqueId string, pageIndex, totalPages int) []gotgbot.InlineKeyboardButton {
	btns := make([]gotgbot.InlineKeyboardButton, 0, 3)
	if pageIndex != 0 {
		btns = append(btns, backButton(uniqueId, pageIndex-1))
	}

	btns = append(btns, gotgbot.InlineKeyboardButton{Text: fmt.Sprintf("💎 %d / %d", pageIndex+1, totalPages), CallbackData: "ignore"})

	if pageIndex+1 != totalPages {
		btns = append(btns, nextButton(uniqueId, pageIndex+1))
	}

	return btns
}

func backButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "« ʙᴀᴄᴋ", CallbackData: fmt.Sprintf("navg|%s_%d", uniqueId, pageIndex), Style: "primary"}
}

func nextButton(uniqueId string, pageIndex int) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: "ɴᴇxᴛ »", CallbackData: fmt.Sprintf("navg|%s_%d", uniqueId, pageIndex), Style: "primary"}
}

// SeasonListCallback handles returning to the season selection list.
func SeasonListCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery
	callbackData := callbackdata.FromString(c.Data)
	if len(callbackData.Args) < 1 {
		return nil
	}

	uniqueId := callbackData.Args[0]
	res, ok, err := _app.Cache.Autofilter.Get(uniqueId)
	if err != nil || !ok {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Session Expired!", ShowAlert: true})
		return nil
	}

	if c.From.Id != res.FromUser {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Not your session!", ShowAlert: true})
		return nil
	}

	buttons := res.Files[0].ProcessSeasons(uniqueId)
	text := "<b>Select Season:</b>"
	_, _, err = c.Message.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	if err == nil {
		middleware.ReactWithRandomEmoji(bot, c.Message.GetChat().Id, c.Message.GetMessageId(), _app.Config, _app.Log)
	}
	return err
}

// SeasonCallback handles clicking on a Season button.
func SeasonCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery
	callbackData := callbackdata.FromString(c.Data)
	if len(callbackData.Args) < 2 {
		return nil
	}

	uniqueId := callbackData.Args[0]
	season, _ := strconv.Atoi(callbackData.Args[1])
	pageIndex := 0
	if len(callbackData.Args) > 2 {
		pageIndex, _ = strconv.Atoi(callbackData.Args[2])
	}

	res, ok, err := _app.Cache.Autofilter.Get(uniqueId)
	if err != nil {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "An Error occurred :\\", ShowAlert: true})
		_app.Log.Warn("sn: result from cache failed", zap.Error(err))
		return nil
	}
	if !ok {
		_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "Session Expired! Please search again.",
			ShowAlert: true,
		})
		return err
	}

	if c.From.Id != res.FromUser {
		_, err := c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "You Can't Use This Button!",
			ShowAlert: true,
		})
		return err
	}

	// Filter and Sort files for the selected season
	var allFiles autofilter.Files
	for _, page := range res.Files {
		allFiles = append(allFiles, page...)
	}
	seasonFiles := allFiles.FilterBySeason(season)
	seasonFiles.SortSeries()

	// Create a sub-result for this season to support ALL/SELECT
	seasonUniqueId := fmt.Sprintf("%s_s%d", uniqueId, season)
	seasonResult := &autofilter.SearchResult{
		UniqueId: seasonUniqueId,
		Query:    fmt.Sprintf("%s - Season %d", res.Query, season),
		FromUser: res.FromUser,
		ChatID:   res.ChatID,
		Files:    []autofilter.Files{seasonFiles}, // Store as one page for simplicity
	}
	_ = _app.Cache.Autofilter.Save(seasonResult)

	// Chunk episodes for pagination
	pageSize := 10 // Episodes per page
	totalPages := (len(seasonFiles) + pageSize - 1) / pageSize
	if pageIndex < 0 {
		pageIndex = 0
	}
	if pageIndex >= totalPages {
		pageIndex = totalPages - 1
	}

	start := pageIndex * pageSize
	end := start + pageSize
	if end > len(seasonFiles) {
		end = len(seasonFiles)
	}
	currentPageEpisodes := seasonFiles[start:end]

	var buttons [][]gotgbot.InlineKeyboardButton
	if ctx.EffectiveChat.Type == "private" {
		// Use seasonUniqueId so ALL/SELECT work only for this season
		buttons = append(buttons, headerRow(seasonUniqueId, 0, true))
	}
	fileButtons := currentPageEpisodes.Process(ctx.EffectiveChat.Id, bot.Username, _app.Config)

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

	// Navigation (Diamonds Style)
	navBtns := make([]gotgbot.InlineKeyboardButton, 0, 3)
	if pageIndex > 0 {
		navBtns = append(navBtns, gotgbot.InlineKeyboardButton{
			Text: "« ʙᴀᴄᴋ", CallbackData: fmt.Sprintf("sn|%s_%d_%d", uniqueId, season, pageIndex-1),
			Style: "primary",
		})
	}
	navBtns = append(navBtns, gotgbot.InlineKeyboardButton{
		Text: fmt.Sprintf("💎 %d / %d", pageIndex+1, totalPages), CallbackData: "ignore",
	})
	if pageIndex+1 < totalPages {
		navBtns = append(navBtns, gotgbot.InlineKeyboardButton{
			Text: "ɴᴇxᴛ »", CallbackData: fmt.Sprintf("sn|%s_%d_%d", uniqueId, season, pageIndex+1),
			Style: "primary",
		})
	}
	buttons = append(buttons, navBtns)

	// Footer Action Row 1 (Links)
	newMoviesBtn := gotgbot.InlineKeyboardButton{Text: "🍿 New Movies", Style: "success", CallbackData: "ignore"}
	if _app.Config.NewMoviesUrl != "" {
		newMoviesBtn.Url = _app.Config.NewMoviesUrl
		newMoviesBtn.CallbackData = ""
	}
	updatesBtn := gotgbot.InlineKeyboardButton{Text: "📺 Updates", Style: "success", CallbackData: "ignore"}
	if _app.Config.UpdatesUrl != "" {
		updatesBtn.Url = _app.Config.UpdatesUrl
		updatesBtn.CallbackData = ""
	}
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{newMoviesBtn, updatesBtn})

	// Footer Action Row 2 (Actions)
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{
		{Text: "📢 Share", SwitchInlineQuery: &res.Query, Style: "success"},
		{Text: "❌ Close", CallbackData: "close", Style: "danger"},
		{Text: "🔙 Back", CallbackData: fmt.Sprintf("af|%s", uniqueId), Style: "primary"},
	})

	_, _, err = c.Message.EditText(bot, "<b>Select Episode:</b>", &gotgbot.EditMessageTextOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	return err
}

// LanguageCallback handles clicking on a Language filter button.
func LanguageCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	c := ctx.CallbackQuery
	callbackData := callbackdata.FromString(c.Data)
	if len(callbackData.Args) < 2 {
		return nil
	}

	uniqueId := callbackData.Args[0]
	language := callbackData.Args[1]

	res, ok, err := _app.Cache.Autofilter.Get(uniqueId)
	if err != nil || !ok {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Session Expired!", ShowAlert: true})
		return nil
	}

	if c.From.Id != res.FromUser {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Not your session!", ShowAlert: true})
		return nil
	}

	// Filter files by language
	var allFiles autofilter.Files
	for _, page := range res.Files {
		allFiles = append(allFiles, page...)
	}
	langFiles := allFiles.FilterByLanguage(language)
	
	if len(langFiles) == 0 {
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "No files found for this language!", ShowAlert: true})
		return nil
	}

	// We'll show up to 10 results for simplicity in filter view
	if len(langFiles) > 10 {
		langFiles = langFiles[:10]
	}

	var buttons [][]gotgbot.InlineKeyboardButton
	fileButtons := langFiles.Process(ctx.EffectiveChat.Id, bot.Username, _app.Config)
	buttons = append(buttons, fileButtons...)

	// Footer Action Row
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{
		{Text: "📢 Share", SwitchInlineQuery: &res.Query, Style: "success"},
		{Text: "❌ Close", CallbackData: "close", Style: "danger"},
		{Text: "🔙 Back", CallbackData: "reset|" + res.Query, Style: "primary"},
	})

	text := fmt.Sprintf("<b>Results for Language:</b> <code>%s</code>\n\n<i>Found %d files.</i>", language, len(langFiles))

	_, _, err = c.Message.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		ParseMode:   gotgbot.ParseModeHTML,
	})
	return err
}
