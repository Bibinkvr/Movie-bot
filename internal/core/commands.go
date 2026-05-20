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
	ok, _ := fsub.CheckFsub(_app, bot, ctx)
	if !ok {
		return nil
	}
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
		firstWord := strings.Fields(text)[0]
		cmd, _, _ := strings.Cut(firstWord, "@")
		commandName = strings.ToLower(strings.TrimPrefix(cmd, "/"))
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
	case "privacy":
		msg = _app.Config.GetPrivacyMessage()
	case "stats": // failsafe
		return Stats(bot, ctx)
	case "top":
		return TopSearching(bot, ctx)
	default:
		msg = &message.Message{
			Text: fmt.Sprintf("Command %v Was Not Found!", commandName),
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
		sentMsg, err = msg.Send(bot, ctx.EffectiveChat.Id)
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

	m := ctx.EffectiveMessage

	limiter.Wait()
	prg, _ := m.Reply(bot, "⏳ 𝖴𝗉𝗅𝗈𝖺𝖽𝗂𝗇𝗀 . . .", nil)

	f, err := os.Open("logs/app.log")
	if err != nil {
		_app.Log.Warn("open log file failed", zap.Error(err))
		return nil
	}

	limiter.Wait()
	_, err = bot.SendDocument(
		ctx.EffectiveChat.Id,
		gotgbot.InputFileByReader("app-log.json", f),
		&gotgbot.SendDocumentOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: m.MessageId,
			},
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{button.Close(m.From.Id)}}},
		},
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
		for lang, val := range s.Languages {
			topLangs.WriteString(fmt.Sprintf("├ %s: %d\n", lang, val))
			count++
			if count >= 5 {
				break
			}
		}
	} else {
		topLangs.WriteString("<i>No language data available</i>")
	}

	var topCountries strings.Builder
	if len(s.Countries) > 0 {
		count := 0
		for country, val := range s.Countries {
			topCountries.WriteString(fmt.Sprintf("├ %s: %d\n", country, val))
			count++
			if count >= 8 {
				break
			}
		}
	} else {
		topCountries.WriteString("<i>No country data available</i>")
	}

	var fsubStatus strings.Builder
	fsubChannels := _app.Config.GetFsubChannels()
	if len(fsubChannels) == 0 {
		fsubStatus.WriteString("<i>Disabled / No Channels</i>")
	} else {
		for i, ch := range fsubChannels {
			analytics, _ := _app.DB.GetFsubAnalytics(ch.ID)
			memberCount, _ := bot.GetChatMemberCount(ch.ID, nil)
			if memberCount > 0 {
				memberCount -= 1 // Exclude the bot
			}

			fsubStatus.WriteString(fmt.Sprintf(`➲ <b>ForceSub %02d</b> | 𝖱𝖾𝗊𝗎𝖾𝗌𝗍 𝖥𝗈𝗋𝖼𝖾𝖲𝗎𝖻 ✅
○ %s | <code>%d</code>
○ <b>Link:</b> %s

○ Total DB Members: %d
• TG Count: %d <i>(Req: %d | Joined: %d)</i>
     DB Count: %d
• Total Requests in Channel: %d

`, i+1, ch.Title, ch.ID, ch.InviteLink, analytics.BotUsers, memberCount+analytics.Requested, analytics.Requested, memberCount, analytics.Requested, analytics.TotalRequests))
		}
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

<b>Activity 24h</b>

<code>%d Users Active</code>

━━━━━━━━━━━━━━━━━━  

<b>Languages</b>

%s
━━━━━━━━━━━━━━━━━━  

<b>Users From</b>

%s

━━━━━━━━━━━━━━━━━━  
<b>Force Subscribe Status</b>
%s`,
		filesCount,
		s.TotalUsers,
		groupsCount,
		mostSearched.String(),
		s.ActiveUsers,
		topLangs.String(),
		topCountries.String(),
		fsubStatus.String(),
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
