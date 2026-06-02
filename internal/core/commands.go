/*
Basic static commands that don't require additional helpers
*/

package core

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"autofilterbot/internal/button"
	"autofilterbot/internal/fsub"
	"autofilterbot/internal/functions"
	"autofilterbot/internal/limiter"
	"autofilterbot/internal/middleware"
	"autofilterbot/internal/model/message"
	"autofilterbot/pkg/callbackdata"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// StaticCommands handles all static text commands like about, help, privacy etc.
// Also handles callback queries in the format cmd:<command_name>
func StaticCommands(bot *gotgbot.Bot, ctx *ext.Context) error {
	var (
		commandName string
		isMedia     bool
	)

	isCallback := ctx.CallbackQuery != nil
	if c := ctx.CallbackQuery; isCallback {
		data := callbackdata.FromString(c.Data)
		if len(data.Path) > 1 {
			commandName = strings.ToLower(data.Path[1])
		}

		if m, ok := c.Message.(gotgbot.Message); ok {
			isMedia = functions.HasMedia(&m)
		}
	} else {
		m := ctx.EffectiveMessage

		text := ctx.EffectiveMessage.GetText()
		if text == "" {
			commandName = "start"
		} else {
			fields := strings.Fields(text)
			if len(fields) > 0 {
				firstWord := fields[0]
				cmd, _, _ := strings.Cut(firstWord, "@")
				commandName = strings.ToLower(strings.TrimPrefix(cmd, "/"))
			} else {
				commandName = "start"
			}
		}
		isMedia = functions.HasMedia(m)
	}

	var (
		msg         *message.Message
		err         error
		extraValues map[string]any
	)

	switch commandName {
	case "start":
		msg = _app.Config.GetStartMessage(bot.Username)
	case "about":
		msg = _app.Config.GetAboutMessage()
		lat := time.Since(time.Unix(ctx.EffectiveMessage.Date, 0))
		extraValues = map[string]any{
			"os":         runtime.GOOS,
			"database":   _app.DB.GetName(),
			"latency":    fmt.Sprintf("%.2fms", float64(lat.Microseconds())/1000.0),
			"go_version": runtime.Version(),
		}
	case "help":
		msg = _app.Config.GetHelpMessage()
	case "movies":
		msg = _app.Config.GetMoviesMessage()
	case "series":
		msg = _app.Config.GetSeriesMessage()
	case "privacy":
		msg = _app.Config.GetPrivacyMessage()
	case "stats": // failsafe
		return Stats(bot, ctx)
	case "fstats":
		return FStats(bot, ctx)
	case "top":
		return TopSearching(bot, ctx)
	default:
		msg = &message.Message{
			Text: fmt.Sprintf("Command %v Was Not Found!", commandName),
		}
	}

	var sendOpts *gotgbot.SendMessageOpts
	if commandName == "start" && !isCallback {
		sendOpts = &gotgbot.SendMessageOpts{
			MessageEffectId: "5046509860340391262", // Confetti Effect
		}
	}

	var sentMsg *gotgbot.Message
	msg.Format(_app.BasicMessageValues(ctx, extraValues))

	if isCallback {
		if isMedia {
			limiter.Wait()
			sentMsg, _, err = ctx.EffectiveMessage.EditCaption(bot, &gotgbot.EditMessageCaptionOpts{
				Caption:     msg.Text,
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: button.UnwrapKeyboard(msg.Keyboard)},
				ParseMode:   gotgbot.ParseModeHTML,
			})
		} else {
			limiter.Wait()
			sentMsg, _, err = ctx.EffectiveMessage.EditText(bot, msg.Text, &gotgbot.EditMessageTextOpts{
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: button.UnwrapKeyboard(msg.Keyboard)},
				ParseMode:   gotgbot.ParseModeHTML,
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
			})
		}
	} else {
		limiter.Wait()
		if sendOpts != nil {
			sentMsg, err = msg.Send(bot, ctx.EffectiveChat.Id, sendOpts)
		} else {
			sentMsg, err = msg.Send(bot, ctx.EffectiveChat.Id)
		}
	}

	if err == nil && sentMsg != nil {
		middleware.ReactWithRandomEmoji(bot, sentMsg.Chat.Id, sentMsg.MessageId, _app.Config, _app.Log)
	}

	if err != nil {
		_app.Log.Warn(err.Error(), zap.String("command", commandName))
	}

	return nil
}

// Logs handles the /logs command.
func Logs(bot *gotgbot.Bot, ctx *ext.Context) error {
	ok, _ := fsub.CheckFsub(_app, bot, ctx)
	if !ok {
		return nil
	}
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	chatId := ctx.EffectiveChat.Id
	var prg *gotgbot.Message
	var err error

	limiter.Wait()
	if ctx.CallbackQuery != nil {
		ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "⏳ Downloading logs..."})
		prg, err = bot.SendMessage(chatId, "⏳ 𝖴𝗉𝗅𝗈𝖺𝖽𝗂𝗇𝗀 . . .", nil)
	} else {
		m := ctx.EffectiveMessage
		prg, err = m.Reply(bot, "⏳ 𝖴𝗉𝗅𝗈𝖺𝖽𝗂𝗇𝗀 . . .", nil)
	}
	if err != nil {
		_app.Log.Warn("logs: failed to send progress message", zap.Error(err))
	}

	f, err := os.Open("logs/app.log")
	if err != nil {
		_app.Log.Warn("open log file failed", zap.Error(err))
		if prg != nil {
			prg.Delete(bot, nil)
		}
		return nil
	}
	defer f.Close()

	sendOpts := &gotgbot.SendDocumentOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{button.Close(ctx.EffectiveUser.Id)}}},
	}
	if ctx.CallbackQuery == nil && ctx.EffectiveMessage != nil {
		sendOpts.ReplyParameters = &gotgbot.ReplyParameters{
			MessageId: ctx.EffectiveMessage.MessageId,
		}
	}

	limiter.Wait()
	_, err = bot.SendDocument(
		chatId,
		gotgbot.InputFileByReader("app-log.json", f),
		sendOpts,
	)
	if err != nil {
		_app.Log.Warn("send log file failed", zap.Error(err))
	}

	if prg != nil {
		prg.Delete(bot, nil)
	}

	return nil
}

// Stats handles the stats command and callback query.
func Stats(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	analytics := _app.Analytics.(*AnalyticsService)
	s, err := analytics.GetStats()
	if err != nil {
		_app.Log.Warn("stats: get stats failed", zap.Error(err))
		return nil
	}

	dbStats, _ := _app.DB.Stats()
	var filesCount string
	var groupsCount int64
	if dbStats != nil {
		filesCount = fmt.Sprint(dbStats.Files)
		groupsCount = dbStats.Groups
	} else {
		filesCount = "0"
	}

	var mostSearched strings.Builder
	if len(s.TopSearches) > 0 {
		for i, st := range s.TopSearches {
			mostSearched.WriteString(fmt.Sprintf("%d. %s (%d)\n", i+1, st.Query, st.Count))
			if i >= 9 {
				break
			}
		}
	} else {
		mostSearched.WriteString("<i>No searches recorded</i>")
	}

	var topLangs strings.Builder
	if len(s.Languages) > 0 {
		count := 0
		type langEntry struct {
			lang string
			val  int64
		}
		var entries []langEntry
		for lang, val := range s.Languages {
			if lang == "Unknown" {
				continue
			}
			entries = append(entries, langEntry{lang, val})
		}
		
		// Sort descending
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].val < entries[j].val {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		for _, entry := range entries {
			topLangs.WriteString(fmt.Sprintf("├ %s: %d\n", entry.lang, entry.val))
			count++
			if count >= 5 {
				break
			}
		}
		if count == 0 {
			topLangs.WriteString("<i>No language data available</i>")
		}
	} else {
		topLangs.WriteString("<i>No language data available</i>")
	}

	var topCountries strings.Builder
	if len(s.Countries) > 0 {
		count := 0
		type countryEntry struct {
			country string
			val     int64
		}
		var entries []countryEntry
		for country, val := range s.Countries {
			entries = append(entries, countryEntry{country, val})
		}
		// Sort descending
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].val < entries[j].val {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		for _, entry := range entries {
			topCountries.WriteString(fmt.Sprintf("├ %s: %d\n", entry.country, entry.val))
			count++
			if count >= 8 {
				break
			}
		}
	} else {
		topCountries.WriteString("<i>No country data available</i>")
	}

	text := fmt.Sprintf(`<b>━━ SYSTEM METRICS ━━</b>

<b>Overview</b>

Files     <code>%s</code>  
Users     <code>%d</code>  
Groups    <code>%d</code>  

━━━━━━━━━━━━━━━━━━  

<b>Most Searched</b>

%s
━━━━━━━━━━━━━━━━━━  

<b>Activity</b>

├ Daily: <code>%d</code>
├ Weekly: <code>%d</code>
├ Monthly: <code>%d</code>

━━━━━━━━━━━━━━━━━━  

<b>Languages</b>

%s
━━━━━━━━━━━━━━━━━━  

<b>Users From</b>

%s`,
		filesCount,
		s.TotalUsers,
		groupsCount,
		mostSearched.String(),
		s.ActiveUsers,
		s.ActiveUsersWeekly,
		s.ActiveUsersMonthly,
		topLangs.String(),
		topCountries.String(),
	)

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{
			Text: "🔄 Refresh", CallbackData: "cmd:stats", Style: "primary",
		}, {
			Text: "🔙 Back", CallbackData: "admin:back", Style: "primary",
		}, {
			Text: "🗑️ Close", CallbackData: "close", Style: "danger",
		}}},
	}

	switch {
	case ctx.Message != nil:
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	case ctx.CallbackQuery != nil:
		c := ctx.CallbackQuery
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Stats Refreshed!"})
		_, _, err = c.Message.EditText(bot, text, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	}

	if err != nil {
		_app.Log.Warn("stats: send result failed", zap.Error(err))
	}

	return nil
}

// FStats handles the fstats command and callback query.
func FStats(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !_app.AuthAdmin(ctx) {
		return nil
	}

	var fsubStatus strings.Builder
	fsubChannels := _app.Config.GetFsubChannels()
	if len(fsubChannels) == 0 {
		fsubStatus.WriteString("<i>Disabled / No Channels</i>")
	} else {
		for i, ch := range fsubChannels {
			analytics, _ := _app.DB.GetFsubAnalytics(ch.ID)
			memberCount, err := bot.GetChatMemberCount(ch.ID, nil)
			if err != nil {
				memberCount = 0
			} else if memberCount > 0 {
				memberCount -= 1 // Exclude the bot
			}

			fsubStatus.WriteString(fmt.Sprintf(`➲ <b>ForceSub %02d</b> | 𝖱𝖾𝗊𝗎𝖾𝗌𝗍 𝖥𝗈𝗋𝖼𝖾𝖲𝗎𝖻 ✅
○ <b>Title:</b> %s
○ <b>ID:</b> <code>%d</code>
○ <b>Link:</b> %s

• <b>Joined Members (TG):</b> %d
• <b>Pending Requests (DB):</b> %d <i>(Bot Users: %d)</i>
  ├ Daily: <code>%d</code>
  ├ Weekly: <code>%d</code>
  ├ Monthly: <code>%d</code>
• <b>Total FSub Reach:</b> %d

`, i+1, ch.Title, ch.ID, ch.InviteLink, memberCount, analytics.TotalRequests, analytics.Requested, analytics.DailyRequests, analytics.WeeklyRequests, analytics.MonthlyRequests, memberCount+analytics.TotalRequests))
		}
	}

	text := fmt.Sprintf(`<b>━━ FORCE SUBSCRIBE METRICS ━━</b>

<b>Configured Channels:</b> %d

━━━━━━━━━━━━━━━━━━  

%s`, len(fsubChannels), fsubStatus.String())

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{
			Text: "🔄 Refresh", CallbackData: "fstats", Style: "primary",
		}, {
			Text: "🔙 Back", CallbackData: "admin:back", Style: "primary",
		}, {
			Text: "🗑️ Close", CallbackData: "close", Style: "danger",
		}}},
	}

	var err error
	switch {
	case ctx.Message != nil:
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	case ctx.CallbackQuery != nil:
		c := ctx.CallbackQuery
		c.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "FSub Stats Refreshed!"})
		_, _, err = c.Message.EditText(bot, text, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   gotgbot.ParseModeHTML,
		})
	}

	if err != nil {
		_app.Log.Warn("fstats: send result failed", zap.Error(err))
	}

	return nil
}

// IdCommand handles the /id command, replying with the chat ID or user ID
func IdCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	m := ctx.EffectiveMessage
	if m == nil {
		return nil
	}

	var text string

	// Handle forwarded messages
	if m.ForwardOrigin != nil {
		mo := m.ForwardOrigin.MergeMessageOrigin()
		if mo.Chat != nil {
			text = fmt.Sprintf("<b>Forwarded Channel ID:</b> <code>%d</code>", mo.Chat.Id)
		} else if mo.SenderChat != nil {
			text = fmt.Sprintf("<b>Forwarded Chat ID:</b> <code>%d</code>", mo.SenderChat.Id)
		} else if mo.SenderUser != nil {
			text = fmt.Sprintf("<b>Forwarded User ID:</b> <code>%d</code>", mo.SenderUser.Id)
		}
	}

	// Handle normal messages
	if text == "" {
		if m.Chat.Type == "channel" {
			text = fmt.Sprintf("<b>Channel ID:</b> <code>%d</code>", m.Chat.Id)
		} else if m.Chat.Type == "private" {
			text = fmt.Sprintf("<b>User ID:</b> <code>%d</code>", m.From.Id)
		} else {
			text = fmt.Sprintf("<b>Chat ID:</b> <code>%d</code>\n<b>User ID:</b> <code>%d</code>", m.Chat.Id, m.From.Id)
		}
	}

	_, err := m.Reply(bot, text, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
	return err
}
