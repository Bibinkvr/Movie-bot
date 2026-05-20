package middleware

import (
	"math/rand"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

// ConfigProvider is an interface that allows the middleware to access the bot's configuration.
type ConfigProvider interface {
	GetEmojiMode() bool
	GetReactions() []string
}

// ReactWithRandomEmoji is a helper function that reacts to a message with a random emoji from the config.
func ReactWithRandomEmoji(bot *gotgbot.Bot, chatId, msgId int64, config ConfigProvider, log *zap.Logger) {
	if !config.GetEmojiMode() {
		return
	}

	reactions := config.GetReactions()
	if len(reactions) == 0 {
		reactions = []string{"👍", "🔥", "❤️", "🥰", "👏", "😁", "🤔", "🤯", "😱", "🎉", "🤩", "🙏", "🤣", "😍", "⚡", "😈"}
	}

	emoji := reactions[rand.Intn(len(reactions))]

	go func() {
		log.Info("Attempting reaction", zap.Int64("chat_id", chatId), zap.Int64("msg_id", msgId), zap.String("emoji", emoji))
		_, err := bot.SetMessageReaction(chatId, msgId, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{
				gotgbot.ReactionTypeEmoji{Emoji: emoji},
			},
			IsBig: true,
		})
		if err != nil {
			log.Warn("failed to set message reaction", zap.Error(err), zap.String("emoji", emoji), zap.Int64("chat_id", chatId))
		}
	}()
}

// AutoReaction is a middleware that automatically reacts to incoming messages.
func AutoReaction(config ConfigProvider, log *zap.Logger) func(bot *gotgbot.Bot, ctx *ext.Context) error {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		if !config.GetEmojiMode() {
			return nil
		}

		if ctx.Message == nil {
			return nil
		}

		log.Info("AutoReaction called", zap.Int64("chat_id", ctx.Message.Chat.Id), zap.Int64("message_id", ctx.Message.MessageId), zap.String("text", ctx.Message.Text))

		ReactWithRandomEmoji(bot, ctx.Message.Chat.Id, ctx.Message.MessageId, config, log)
		return nil
	}
}
