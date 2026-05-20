package configpanel

import (
	"fmt"

	"autofilterbot/pkg/panel"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

// BoolField is a helper for modifying bool fields.
func BoolField(app AppPreview, fieldName string, description ...string) panel.CallbackFunc {
	return func(ctx *panel.Context) (string, [][]gotgbot.InlineKeyboardButton, error) {
		var (
			op   string
			data = ctx.CallbackData
		)

		if len(data.Args) != 0 {
			op = data.Args[0]
		}

		var s string

		switch op {
		case OperationSet:
			err := app.GetDB().UpdateConfig(ctx.Bot.Id, fieldName, true)
			if err != nil {
				return "", nil, err
			}

			s = fmt.Sprintf("<i><b>✅ %s has been Enabled !</b></i>", ctx.Page.DisplayName)
		case OperationReset:
			err := app.GetDB().ResetConfig(ctx.Bot.Id, fieldName)
			if err != nil {
				return "", nil, err
			}

			s = fmt.Sprintf("<i><b>✅ %s has been Reset !</b></i>", ctx.Page.DisplayName)
		default:
			var s string
			if len(description) != 0 {
				s = "ℹ️ " + description[0]
			}

			return s + fmt.Sprintf("<i>Use The Buttons Below to Enable/Disable %s</i>", ctx.Page.DisplayName),
				[][]gotgbot.InlineKeyboardButton{{{Text: "ᴇɴᴀʙʟᴇ ✅", CallbackData: data.RemoveArgs().AddArg(OperationSet).ToString(), Style: "success"}, {Text: "ᴅɪsᴀʙʟᴇ ❌", CallbackData: data.RemoveArgs().AddArg(OperationReset).ToString(), Style: "danger"}}},
				nil
		}

		go app.RefreshConfig() // is a goroutine a bit overkill here

		return s, nil, nil
	}
}
