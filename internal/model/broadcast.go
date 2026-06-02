package model

import (
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type BroadcastMessage struct {
	UserId    int64 `json:"user_id" bson:"user_id"`
	MessageId int64 `json:"message_id" bson:"message_id"`
}

type Broadcast struct {
	ID             string                         `json:"_id" bson:"_id"`
	Text           string                         `json:"text" bson:"text"`
	FileId         string                         `json:"file_id" bson:"file_id"`
	Method         string                         `json:"method" bson:"method"`
	InlineKeyboard [][]gotgbot.InlineKeyboardButton `json:"inline_keyboard,omitempty" bson:"inline_keyboard,omitempty"`
	CreatedAt      time.Time                      `json:"created_at" bson:"created_at"`
	Status         string                         `json:"status" bson:"status"` // "pending", "sending", "completed", "cancelled", "messages_deleted"
	Total          int                            `json:"total" bson:"total"`
	Success        int                            `json:"success" bson:"success"`
	Failed         int                            `json:"failed" bson:"failed"`
	Blocked        int                            `json:"blocked" bson:"blocked"`
	Deleted        int                            `json:"deleted" bson:"deleted"`
	OtherErr       int                            `json:"other_err" bson:"other_err"`
	SentMessages   []BroadcastMessage             `json:"sent_messages,omitempty" bson:"sent_messages,omitempty"`
}
