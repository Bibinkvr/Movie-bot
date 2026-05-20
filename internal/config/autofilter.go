package config

const (
	defaultMaxResults = 50
	defaultMaxPerPage = 10
	defaultMaxPages   = 5
)

func (c *Config) GetMaxResults() int {
	if c.MaxResults != 0 {
		return c.MaxResults
	}

	return defaultMaxResults
}

func (c *Config) GetMaxPerPage() int {
	if c.MaxPerPage != 0 {
		return c.MaxPerPage
	}

	return defaultMaxPerPage
}

func (c *Config) GetMaxPages() int {
	if c.MaxResults != 0 {
		return c.MaxResults
	}

	return defaultMaxPages
}

func (c *Config) GetResultTemplate() string {
	if c.ResultTemplate != "" {
		return c.ResultTemplate
	}

	return `<b>🍿 Hᴇʏ <tg-spoiler>{mention}</tg-spoiler>, I'ᴠᴇ Fᴏᴜɴᴅ Sᴏᴍᴇ ᴍᴀᴛᴄʜᴇs ғᴏʀ ʏᴏᴜ!</b>
<blockquote><b>🔍 Sᴇᴀʀᴄʜ Query:</b> <code>{query}</code>
<b>📂 Tᴏᴛᴀʟ Fɪʟᴇs Fᴏᴜɴᴅ:</b> <code>{total}</code></blockquote>

{warn}`
}

func (c *Config) GetNoResultText() string {
	if c.NoResultText != "" {
		return c.NoResultText
	}

	return `<b>💔 Oᴏᴘs {mention}, ɴᴏ ʀᴇsᴜʟᴛs ᴡᴇʀᴇ ғᴏᴜɴᴅ...</b>
<blockquote><b>🔍 Query:</b> <code>{query}</code>
<i>⚠️ Pʟᴇᴀsᴇ ᴄʜᴇᴄᴋ sᴘᴇʟʟɪɴɢ ᴏʀ ᴛʀʏ ᴅɪғғᴇʀᴇɴᴛ ᴋᴇʏᴡᴏʀᴅs.</i></blockquote>

<i>💡 Tɪᴘ: Yᴏᴜ ᴄᴀɴ ᴜsᴇ ᴛʜᴇ sᴜɢɢᴇsᴛɪᴏɴ ʙᴜᴛᴛᴏɴ ʙᴇʟᴏᴡ 👇</i>`
}

func (c *Config) GetButtonTemplate() string {
	if c.ButtonTemplate != "" {
		return c.ButtonTemplate
	}

	return "📂 {file_size} {file_name}"
}

func (c *Config) GetSizeButton() bool {
	return c.SizeButton
}
