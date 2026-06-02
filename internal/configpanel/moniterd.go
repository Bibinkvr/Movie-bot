package configpanel

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"autofilterbot/internal/config"
	"autofilterbot/pkg/conversation"
	"autofilterbot/pkg/panel"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/pkg/errors"
)

func MonitoredChannelsField(app AppPreview) panel.CallbackFunc {
	return func(ctx *panel.Context) (string, [][]gotgbot.InlineKeyboardButton, error) {
		var (
			op   string
			data = ctx.CallbackData
		)

		if len(data.Args) != 0 {
			op = data.Args[0]
		}

		currentChannels := app.GetConfig().FileChannels

		switch op {
		case OperationDelete:
			if len(data.Args) < 2 {
				return "", nil, errors.New("configpanel: moniterd: insufficient data for delete operation")
			}

			channelID, err := strconv.ParseInt(data.Args[1], 10, 64)
			if err != nil {
				return "", nil, err
			}

			for i, id := range currentChannels {
				if id == channelID {
					currentChannels = slices.Delete(currentChannels, i, i+1)

					app.GetDB().UpdateConfig(ctx.Bot.Id, "file_channels", currentChannels)
					go app.RefreshConfig()

					return "Monitored Channel was Deleted Successfully ✅", nil, nil
				}
			}

			return "Monitored Channel to Delete Was not Found 🫤", nil, nil

		case OperationReset:
			conv := conversation.NewConversatorFromUpdate(ctx.Bot, ctx.Update.Update)

			m, err := conv.Ask(app.GetContext(), "Are you sure you want to clear all Monitored Channels? (y/N)", nil)
			if err != nil {
				return "", nil, errors.Wrap(err, "configpanel: moniterd: send reset confirmation message failed")
			}

			if strings.ToLower(m.Text) != "y" {
				return "Reset Operation Cancelled!", nil, nil
			}

			app.GetDB().UpdateConfig(ctx.Bot.Id, "file_channels", []int64{})
			go app.RefreshConfig()

			return "Monitored Channels Have Been Cleared Succesfully ✅", nil, nil

		case OperationSet:
			chatIDStr, _ := data.GetArg(1)
			if chatIDStr == "" {
				conv := conversation.NewConversatorFromUpdate(ctx.Bot, ctx.Update.Update)

				m, err := conv.Ask(app.GetContext(), "Please Forward a Post from the Channel (with quotes) or Send the Chat ID.\n\n<i>You can send multiple IDs separated by spaces!</i>", nil)
				if err != nil {
					return "", nil, errors.Wrap(err, "configpanel: moniterd: send channel request message failed")
				}
				if m.ForwardOrigin != nil {
					if f, ok := m.ForwardOrigin.(gotgbot.MessageOriginChannel); ok {
						chatIDStr = fmt.Sprint(f.Chat.Id)
					}
				} else {
					chatIDStr = strings.TrimSpace(m.Text)
				}
			}

			if chatIDStr == "" {
				return "Message was not forwarded from a channel nor contains a channel ID!", nil, nil
			}

			ids := strings.Fields(chatIDStr)
			var addedCount int
			var lastResult string

			for _, idStr := range ids {
				chatID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					lastResult = fmt.Sprintf("Invalid Chat ID format: %s", idStr)
					continue
				}

				chat, err := ctx.Bot.GetChat(chatID, nil)
				if err != nil {
					lastResult = fmt.Sprintf("Failed to get chat for %d: %s. Make sure the bot is added to the channel!", chatID, err.Error())
					continue
				}

				isDuplicate := false
				for _, id := range currentChannels {
					if id == chat.Id {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					lastResult = fmt.Sprintf("Channel %s is already monitored!", chat.Title)
					continue
				}

				currentChannels = append(currentChannels, chat.Id)
				addedCount++
				lastResult = fmt.Sprintf("%s added successfully ✅", chat.Title)

				ctx.Bot.SendMessage(ctx.CallbackQuery.From.Id, lastResult, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
			}

			if addedCount > 0 {
				app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameFileChannels, currentChannels)
				go app.RefreshConfig()
				return fmt.Sprintf("%d channel(s) added successfully! 🚀", addedCount), nil, nil
			}

			return lastResult, nil, nil

		default:
			var s strings.Builder
			s.WriteString("ℹ️ <i>Monitored Channels are channels where the bot automatically indexes new movies/files when they are posted.</i>\n\n")
			s.WriteString(`<b><u>Options</u></b>
<b>Add</b> - Add a new monitored channel
<b>Delete</b> - Stop monitoring a channel
<b>Reset</b> - Clear all monitored channels`)

			var keyboard [][]gotgbot.InlineKeyboardButton

			for _, id := range currentChannels {
				title := fmt.Sprintf("ID: %d", id)
				chat, err := ctx.Bot.GetChat(id, nil)
				if err == nil {
					title = chat.Title
				}

				keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: title, CallbackData: "ignore"}})
				keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
					{Text: "ᴅᴇʟᴇᴛᴇ 🗑️", CallbackData: ctx.CallbackData.AddArgs(OperationDelete, fmt.Sprint(id)).ToString(), Style: "danger"},
				})
			}

			keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
				{Text: "ᴄʟᴇᴀʀ ⏪", CallbackData: ctx.CallbackData.AddArg(OperationReset).ToString(), Style: "danger"},
				{Text: "ᴀᴅᴅ ➕", CallbackData: ctx.CallbackData.AddArgs(OperationSet).ToString(), Style: "success"},
			})

			return s.String(), keyboard, nil
		}
	}
}
