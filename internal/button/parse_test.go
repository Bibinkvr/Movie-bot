package button

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFromTextWithOptions(t *testing.T) {
	text := "Check out these buttons:\n[Google](url:https://google.com|style:primary|emoji:5373141891321699086)\n[Help](cmd:help|style:success)\n[Start](cmd:start|emoji:🚀)"
	remText, buttons, err := ParseFromText(text)
	assert.NoError(t, err)
	assert.Equal(t, "Check out these buttons:", remText)
	assert.Len(t, buttons, 3)

	// First row, first button
	assert.Equal(t, "Google", buttons[0][0].Text)
	assert.Equal(t, "https://google.com", buttons[0][0].Url)
	assert.Equal(t, "primary", buttons[0][0].Style)
	assert.Equal(t, "5373141891321699086", buttons[0][0].IconCustomEmojiId)

	// Second row, first button
	assert.Equal(t, "Help", buttons[1][0].Text)
	assert.Equal(t, "cmd:help", buttons[1][0].CallbackData)
	assert.Equal(t, "success", buttons[1][0].Style)
	assert.Empty(t, buttons[1][0].IconCustomEmojiId)

	// Third row, first button
	assert.Equal(t, "🚀 Start", buttons[2][0].Text)
	assert.Equal(t, "cmd:start", buttons[2][0].CallbackData)
	assert.Empty(t, buttons[2][0].IconCustomEmojiId)
}
