package configpanel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"autofilterbot/internal/config"
	"autofilterbot/pkg/conversation"
	"autofilterbot/pkg/panel"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/pkg/errors"
)

type getChatResponse struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

// ResultsChannelField creates the settings page callback function for the Results Channel.
func ResultsChannelField(app AppPreview) panel.CallbackFunc {
	return func(ctx *panel.Context) (string, [][]gotgbot.InlineKeyboardButton, error) {
		var (
			op   string
			data = ctx.CallbackData
		)

		if len(data.Args) != 0 {
			op = data.Args[0]
		}

		switch op {
		case OperationSet:
			conv := conversation.NewConversatorFromUpdate(ctx.Bot, ctx.Update.Update)
			m, err := conv.Ask(app.GetContext(), "Please send the Channel Username (e.g. <code>@MyChannel</code>) or ID (e.g. <code>-1001234567890</code>) where search results will be posted:\n\n<i>Note: Make sure the bot is an admin in the channel first!</i>", nil)
			if err != nil {
				return "", nil, errors.Wrap(err, "configpanel: results_channel: ask failed")
			}

			val := strings.TrimSpace(m.Text)
			if val == "" {
				return "Operation Cancelled (Empty Input)", nil, nil
			}

			// Resolve username/ID to Chat ID & username
			var resolvedID int64
			var resolvedUsername string

			// Try to parse as ID first
			chatID, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				// Call GetChat to verify
				chat, err := ctx.Bot.GetChat(chatID, nil)
				if err != nil {
					return fmt.Sprintf("❌ Error: Could not resolve channel ID %d. Make sure the bot is added as admin in that channel.", chatID), nil, nil
				}
				resolvedID = chat.Id
				resolvedUsername = chat.Username
			} else {
				// Must be a username
				username := val
				if !strings.HasPrefix(username, "@") {
					username = "@" + username
				}
				
				// Unfortunately, gotgbot GetChat takes int64. But we can use Request for username!
				// Actually wait, let me just use the standard Request without unmarshaling if possible.
				// For now, let's keep the Request for username since GetChat needs int64.
				
				resBytes, err := ctx.Bot.Request("getChat", map[string]any{"chat_id": username}, nil)
				if err != nil {
					return fmt.Sprintf("❌ Error: Could not resolve channel username %s. Make sure the username is correct and the bot is added as admin.", username), nil, nil
				}
				
				var resp struct {
					Id       int64  `json:"id"`
					Username string `json:"username"`
				}
				if err := json.Unmarshal(resBytes, &resp); err != nil {
					return "❌ Error: Failed to parse chat info response.", nil, nil
				}
				resolvedID = resp.Id
				resolvedUsername = resp.Username
			}

			// Update in DB
			err = app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameResultsChannel, val)
			if err != nil {
				return "", nil, err
			}
			err = app.GetDB().UpdateConfig(ctx.Bot.Id, config.FieldNameResultsChannelID, resolvedID)
			if err != nil {
				return "", nil, err
			}

			go app.RefreshConfig()

			successMsg := fmt.Sprintf("<i><b>✅ Results Channel configured successfully!</b></i>\n\n<b>ID:</b> <code>%d</code>", resolvedID)
			if resolvedUsername != "" {
				successMsg += fmt.Sprintf("\n<b>Username:</b> @%s", resolvedUsername)
			}
			return successMsg, nil, nil

		case OperationReset:
			err := app.GetDB().ResetConfig(ctx.Bot.Id, config.FieldNameResultsChannel)
			if err != nil {
				return "", nil, err
			}
			err = app.GetDB().ResetConfig(ctx.Bot.Id, config.FieldNameResultsChannelID)
			if err != nil {
				return "", nil, err
			}

			go app.RefreshConfig()

			return "<i><b>✅ Results Channel configuration has been Reset!</b></i>", nil, nil

		default:
			var s strings.Builder
			s.WriteString("ℹ️ <i>Results Channel posts all search results to a specific channel and redirects users to get their files from there. This keeps your group clean.</i>\n\n")

			currentVal := app.GetConfig().GetResultsChannel()
			currentID := app.GetConfig().GetResultsChannelID()

			if currentVal != "" {
				s.WriteString(fmt.Sprintf("⭕ <b>Current Setting:</b> <code>%s</code>\n", currentVal))
			}
			if currentID != 0 {
				s.WriteString(fmt.Sprintf("🆔 <b>Resolved Chat ID:</b> <code>%d</code>\n\n", currentID))
			} else {
				s.WriteString("❌ <b>Status:</b> Not configured (Results are sent directly to group chats).\n\n")
			}

			s.WriteString("<i><b>📝 Click 'Set' to configure or 'Reset' to clear.</b></i>")

			keyboard := [][]gotgbot.InlineKeyboardButton{
				{
					{Text: "sᴇᴛ ✍️", CallbackData: ctx.CallbackData.AddArg(OperationSet).ToString(), Style: "primary"},
					{Text: "ʀᴇsᴇᴛ 🔁", CallbackData: ctx.CallbackData.AddArg(OperationReset).ToString(), Style: "danger"},
				},
			}

			return s.String(), keyboard, nil
		}
	}
}
