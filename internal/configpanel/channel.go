package configpanel

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"autofilterbot/internal/config"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/conversation"
	"autofilterbot/pkg/panel"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ChannelFieldOpts provides optional parameters to ChannelField.
type ChannelFieldOpts struct {
	// Description for the field.
	Description string
	// Maximum number of channels allowed.
	MaxAmount int
	// Indicates whether the user should be asked if a request invite link should be generated.
	AllowRequestInvite bool
}

func ChannelField(app AppPreview, fieldName string, opts ChannelFieldOpts) panel.CallbackFunc {
	return func(ctx *panel.Context) (string, [][]gotgbot.InlineKeyboardButton, error) {
		var (
			op   string
			data = ctx.CallbackData
		)

		if len(data.Args) != 0 {
			op = data.Args[0]
		}

		currentChannels := app.GetConfig().GetFsubChannels()

		switch op {
		case OperationDelete:
			if len(data.Args) < 2 {
				return "", nil, errors.New("configpanel: channel: insufficient data for delete operation")
			}

			channelID, err := strconv.ParseInt(data.Args[1], 10, 64)
			if err != nil {
				return "", nil, err
			}

			for i, c := range currentChannels {
				if c.ID == channelID {
					currentChannels = slices.Delete(currentChannels, i, i+1)

					app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameFsub, currentChannels)
					go app.RefreshConfig()

					return fieldName + " Channel was Deleted Successfully ✅", nil, nil
				}
			}

			return fieldName + " Channel to Delete Was not Found 🫤", nil, nil
		case OperationReset:
			conv := conversation.NewConversatorFromUpdate(ctx.Bot, ctx.Update.Update)

			m, err := conv.Ask(app.GetContext(), "Are you sure you want to delete all Channels? (y/N)", nil)
			if err != nil {
				return "", nil, errors.Wrap(err, "configpanel: channel: send reset confirmation message failed")
			}

			if strings.ToLower(m.Text) != "y" {
				return "Reset Operation Cancelled!", nil, nil
			}

			app.GetDB().ResetConfig(ctx.Bot.Id, config.FieldNameFsub)
			go app.RefreshConfig()

			return fieldName + " Channels Have Been Reset Succesfully ✅", nil, nil
		case OperationSet:
			if opts.MaxAmount != 0 && len(currentChannels) >= opts.MaxAmount {
				ctx.CallbackQuery.Answer(ctx.Bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Channel Limit Reached.\n\nPlease delete a value to try again.", ShowAlert: true})
				return "", nil, nil
			}

			chatIDStr, _ := data.GetArg(1)
			if chatIDStr == "" {
				conv := conversation.NewConversatorFromUpdate(ctx.Bot, ctx.Update.Update)

				m, err := conv.Ask(app.GetContext(), "Please Forward a Post from the Channel (with quotes) or Send the Chat ID(s).\n\n<i>You can send multiple IDs separated by spaces!</i>", nil)
				if err != nil {
					return "", nil, errors.Wrap(err, "configpanel: channel: send channel request message failed")
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

			// Split by spaces to handle multiple IDs
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
					lastResult = fmt.Sprintf("Failed to get chat for %d: %s", chatID, err.Error())
					continue
				}

				isDuplicate := false
				for _, c := range currentChannels {
					if c.ID == chat.Id {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					lastResult = fmt.Sprintf("Channel %s is already added!", chat.Title)
					continue
				}

				isRequest := true

				link, err := ctx.Bot.CreateChatInviteLink(chat.Id, &gotgbot.CreateChatInviteLinkOpts{Name: fieldName, CreatesJoinRequest: isRequest})
				if err != nil {
					lastResult = fmt.Sprintf("Failed to create invite link for %s. Make sure bot is admin!", chat.Title)
					continue
				}

				currentChannels = append(currentChannels, model.Channel{
					ID:                 chat.Id,
					Title:              chat.Title,
					InviteLink:         link.InviteLink,
					CreatesJoinRequest: isRequest,
				})
				addedCount++
				lastResult = fmt.Sprintf("%s added successfully ✅", chat.Title)

				// Send immediate feedback
				ctx.Bot.SendMessage(ctx.CallbackQuery.From.Id, lastResult, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
			}

			if addedCount > 0 {
				app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameFsub, currentChannels)
				go app.RefreshConfig()
				return fmt.Sprintf("%d channel(s) added successfully! 🚀", addedCount), nil, nil
			}

			return lastResult, nil, nil
		case OperationRefresh:
			if len(data.Args) < 2 {
				return "", nil, errors.New("configpanel: channel: insufficient data for refresh operation")
			}

			channelID, err := strconv.ParseInt(data.Args[1], 10, 64)
			if err != nil {
				return "", nil, err
			}

			chat, err := ctx.Bot.GetChat(channelID, nil)
			if err != nil {
				return "", nil, err
			}

			var channelIndex *int

			for i, c := range currentChannels {
				if c.ID != channelID {
					continue
				}

				channelIndex = &i
			}

			if channelIndex == nil {
				return "Channel was not found in saved channels!", nil, nil
			}

			link, err := ctx.Bot.CreateChatInviteLink(chat.Id, &gotgbot.CreateChatInviteLinkOpts{Name: fieldName, CreatesJoinRequest: true})
			if err != nil {
				app.GetLog().Debug("configpanel: channel: failed to generate invite link", zap.Int64("id", chat.Id), zap.Error(err))
				return "Failed to Create Invite Link. Please Make Sure the bot has Permissions to Add Users", nil, nil
			}

			currentChannels[*channelIndex].Title = chat.Title
			currentChannels[*channelIndex].InviteLink = link.InviteLink
			currentChannels[*channelIndex].CreatesJoinRequest = true

			app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameFsub, currentChannels)
			go app.RefreshConfig()

			return "Channel Information has been Updated Successfully ✅", nil, nil
		default:
			var s strings.Builder

			if opts.Description != "" {
				s.WriteString("ℹ️ <i>" + opts.Description + "</i>\n\n")
			}

			s.WriteString(`<b><u>Options</u></b>
<b>Refresh</b> - Refresh channel information (title and invite link)
<b>Add</b> - Add a new channel
<b>Delete</b> - Delete a single channel.
<b>Reset</b> - Reset to default`)

			if opts.MaxAmount != 0 {
				s.WriteString(fmt.Sprintf("\n\n<b>🗒️ Upto %d channel(s) can be added.</b>", opts.MaxAmount))
			}

			var keybaord [][]gotgbot.InlineKeyboardButton

			for _, c := range currentChannels {
				keybaord = append(keybaord, []gotgbot.InlineKeyboardButton{{Text: c.Title, Url: c.InviteLink}})
				keybaord = append(keybaord, []gotgbot.InlineKeyboardButton{
					{Text: "ᴅᴇʟᴇᴛᴇ 🗑️", CallbackData: ctx.CallbackData.AddArgs(OperationDelete, fmt.Sprint(c.ID)).ToString(), Style: "danger"},
					{Text: "ʀᴇғʀᴇsʜ 🔄", CallbackData: ctx.CallbackData.AddArgs(OperationRefresh, fmt.Sprint(c.ID)).ToString(), Style: "primary"},
				})
			}

			keybaord = append(keybaord, []gotgbot.InlineKeyboardButton{
				{Text: "ʀᴇsᴇᴛ ⏪", CallbackData: ctx.CallbackData.AddArg(OperationReset).ToString(), Style: "danger"},
				{Text: "ᴀᴅᴅ ➕", CallbackData: ctx.CallbackData.AddArgs(OperationSet).ToString(), Style: "success"},
			})

			return s.String(), keybaord, nil
		}
	}
}
