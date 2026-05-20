package configpanel

import (
	"fmt"
	"strings"

	"autofilterbot/pkg/conversation"
	"autofilterbot/pkg/panel"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/pkg/errors"
)

// StringFieldOpts provides optional parameters to StringField.
type StringFieldOpts struct {
	// Description for the field.
	Description string
	// Placeholder for the input request.
	Placeholder string
}

// StringField is a helper for configuring string values in the config panel.
func StringField(app AppPreview, fieldName string, opts StringFieldOpts) panel.CallbackFunc {
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

			prompt := opts.Placeholder
			if prompt == "" {
				prompt = fmt.Sprintf("Please send the new value for <b>%s</b>:", ctx.Page.DisplayName)
			}

			m, err := conv.Ask(app.GetContext(), prompt, nil)
			if err != nil {
				return "", nil, errors.Wrap(err, "configpanel: string: ask failed")
			}

			val := strings.TrimSpace(m.Text)
			if val == "" {
				return "Operation Cancelled (Empty Input)", nil, nil
			}

			err = app.GetDB().UpdateConfig(ctx.Bot.Id, fieldName, val)
			if err != nil {
				return "", nil, err
			}

			go app.RefreshConfig()

			return fmt.Sprintf("<i><b>✅ %s has been set successfully!</b></i>", ctx.Page.DisplayName), nil, nil

		case OperationReset:
			err := app.GetDB().ResetConfig(ctx.Bot.Id, fieldName)
			if err != nil {
				return "", nil, err
			}

			go app.RefreshConfig()

			return fmt.Sprintf("<i><b>✅ %s has been Reset!</b></i>", ctx.Page.DisplayName), nil, nil

		default:
			var s strings.Builder

			if opts.Description != "" {
				s.WriteString(fmt.Sprintf("ℹ️ %s\n\n", opts.Description))
			}

			if v, ok := app.GetConfig().ToMap()[fieldName]; ok {
				if str, ok := v.(string); ok && str != "" {
					s.WriteString(fmt.Sprintf("⭕ Current Value: <code>%s</code>\n\n", str))
				}
			}

			s.WriteString(fmt.Sprintf("<i><b>📝 Click 'Set' to update %s or 'Reset' to clear it.</b></i>", ctx.Page.DisplayName))

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
