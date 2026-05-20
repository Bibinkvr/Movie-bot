package functions

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MessageLink is parsed data from a message link.
type MessageLink struct {
	ChatId    int64
	MessageId int64
	Username  string
}

// GetChat gets a chat using it's id or username.
func (m *MessageLink) GetChat(bot *gotgbot.Bot) (*gotgbot.ChatFullInfo, error) {
	switch {
	case m.ChatId != 0:
		return bot.GetChat(m.ChatId, nil)
	case m.Username != "":
		return GetChatFromUsername(bot, m.Username)
	default:
		return nil, errors.New("both id and username are empty")
	}
}

// ParseMessageLink parses a message link in the format t.me/c/<id>/<mid> or t.me/<username>/<mid>.
func ParseMessageLink(s string) (*MessageLink, error) {
	if len(s) < 5 {
		return nil, errors.New("link too short")
	}

	if !strings.Contains(s, "t.me/") && !strings.Contains(s, "telegram.me/") {
		return nil, errors.New("not a telegram link")
	}

	// Clean protocol if present
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")

	split := strings.Split(s, "/")
	if len(split) < 3 {
		return nil, errors.New("invalid link format: not enough segments")
	}

	// Last segment is always Message ID
	msgIdStr := split[len(split)-1]
	msgId, err := strconv.ParseInt(msgIdStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid message id: %s", msgIdStr)
	}

	var username string
	var chatId int64

	// Format: t.me/c/12345/678 (split len 4) or t.me/username/678 (split len 3)
	if split[1] == "c" && len(split) >= 4 {
		// Private channel format: t.me/c/CHATID/MSGID
		chatId, err = strconv.ParseInt(split[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid private chat id: %s", split[2])
		}
	} else if len(split) >= 3 {
		// Public channel format: t.me/USERNAME/MSGID
		target := split[1]
		chatId, err = strconv.ParseInt(target, 10, 64)
		if err != nil {
			// Not a numeric ID, must be a username
			username = target
		}
	}

	if chatId != 0 {
		// Ensure private chat IDs are correctly formatted with -100 prefix if needed
		// But usually mtproto IDs in links are 12345, converted to -10012345
		if chatId > 0 {
			chatId = MtprotoToChatId(chatId)
		}
	}

	if chatId == 0 && username == "" {
		return nil, errors.New("could not extract chat id or username from link")
	}

	return &MessageLink{
		ChatId:    chatId,
		MessageId: msgId,
		Username:  username,
	}, nil
}

// GetChatFromUsername constructs a getChat request using a username.
func GetChatFromUsername(bot *gotgbot.Bot, username string) (*gotgbot.ChatFullInfo, error) {
	r, err := bot.Request("getChat", map[string]any{"chat_id": "@" + username}, nil)
	if err != nil {
		return nil, err
	}

	var c gotgbot.ChatFullInfo

	err = json.Unmarshal(r, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
