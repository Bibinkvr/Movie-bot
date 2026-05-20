package config

import (
	"fmt"

	"autofilterbot/internal/button"
	"autofilterbot/internal/model/message"
)

// GetStartMessage returns the custom start message if available or the default values.
// botUsername must be provided to create the add to group button.
func (c *Config) GetStartMessage(botUsername string) *message.Message {
	var (
		text    string
		buttons [][]button.InlineKeyboardButton
	)

	addToGroupUrl := fmt.Sprintf("https://t.me/%s?startgroup&admin=delete_messages+pin_messages+invite_users+ban_users+promote_members", botUsername)

	if c.StartText != "" {
		text = c.StartText
	} else {
		text = `<b>Welcome to MovieBot Pro 🎬</b>

Hi {mention}, I am your ultimate companion for high-quality movies and series. 

<blockquote><b>🚀 Fast & Direct</b>
Type the exact title to get instant direct links. No redirects, no ads, no complexity.</blockquote>

<b>✨ Key Features</b>
• ⚡️ <b>Ultra-fast search</b> across 100k+ files
• 📂 <b>Direct download</b> links
• 🍿 <b>Crystal clear</b> quality options
• 🛡 <b>Safe & Ad-free</b> experience

<i>Join our community to stay updated with latest OTT releases and daily additions!</i>`
	}

	if len(c.StartButtons) != 0 {
		buttons = c.StartButtons
	} else {
		buttons = [][]button.InlineKeyboardButton{
			{{Text: "🔰 Add me to your Group 🔰", Url: addToGroupUrl, Style: "primary"}},
			{
				{Text: "Help 📢", CallbackData: "cmd:help", Style: "primary"},
				{Text: "About 📖", CallbackData: "cmd:about", Style: "primary"},
			},
			{{Text: "Top Searching ⭐", CallbackData: "cmd:top", Style: "primary"}},
		}
	}

	return &message.Message{
		Text:     text,
		Keyboard: buttons,
	}
}

func (c *Config) GetAboutMessage() *message.Message {
	var (
		text    string
		buttons [][]button.InlineKeyboardButton
	)

	if c.AboutText != "" {
		text = c.AboutText
	} else {
		text = `<b>[ SYSTEM STATUS ]</b>

> language   : {go_version}  
> status     : active

> os        : <code>{os}</code>  
> database  : <code>{database}</code>  

> version   : <code>v0.5</code>

Latency <code>{latency}</code>`
	}

	if len(c.AboutButtons) != 0 {
		buttons = c.AboutButtons
	} else {
		buttons = [][]button.InlineKeyboardButton{
			{{Text: "« ʙᴀᴄᴋ", CallbackData: "cmd:start", Style: "primary"}, {Text: "sᴛᴀᴛs", CallbackData: "cmd:stats", Style: "primary"}},
		}
	}

	return &message.Message{
		Text:     text,
		Keyboard: buttons,
	}
}

func (c *Config) GetHelpMessage() *message.Message {
	var (
		text    string
		buttons [][]button.InlineKeyboardButton
	)

	if c.HelpText != "" {
		text = c.HelpText
	} else {
		text = `
<b>🖐️ 𝖧𝖾𝗋𝖾'𝗌 𝖳𝗐𝗈 𝖶𝖺𝗒𝗌 𝖸𝗈𝗎 𝖢𝖺𝗇 𝖴𝗌𝖾 𝖬𝖾. . .</b>

✈️ <b>𝖨𝗇𝗅𝗂𝗇𝖾</b> : <i>Just Start Typing my Username into any Chat and get Results On The Fly</i>
✍️ <b>𝖦𝗋𝗈𝗎𝗉</b> : <i>Add me to your Group Chat and Just Send the Name of the File you Want</i>

🤖 <b>User Commands:</b>
/start - check if I'm alive
/about - learn a bit about me
/help - get this message
/stats - some number crushing
/privacy - what data I steal
/uinfo - get user data stored
/id - if you know u know

🍷 <b>Exclusive Commands:</b>
/broadcast - spam users with ads
/settings - customize me
/batch - bunch up messages
/genlink - link to single file
/index - gather up files
/delete - assassinate a file
/deleteall - massacre matching files
`
	}

	if len(c.HelpButtons) != 0 {
		buttons = c.HelpButtons
	} else {
		buttons = [][]button.InlineKeyboardButton{
			{{Text: "« ʙᴀᴄᴋ", CallbackData: "cmd:start", Style: "primary"}, {Text: "ᴘʀɪᴠᴀᴄʏ", CallbackData: "cmd:privacy", Style: "primary"}},
		}
	}

	return &message.Message{
		Text:     text,
		Keyboard: buttons,
	}
}

func (c *Config) GetStatsMessage() *message.Message {
	var (
		text    string
		buttons [][]button.InlineKeyboardButton
	)

	if c.StatsText != "" {
		text = c.StatsText
	} else {
		text = `
╭ ▸ 𝖴𝗌𝖾𝗋𝗌 : <code>{users}</code>
├ ▸ 𝖥𝗂𝗅𝖾𝗌 : <code>{files}</code>
├ ▸ 𝖦𝗋𝗈𝗎𝗉𝗌 : <code>{groups}</code>
╰ ▸ 𝖴𝗉𝗍𝗂𝗆𝖾 : <code>{uptime}</code>
`
	}

	if len(c.StatsButtons) != 0 {
		buttons = c.StatsButtons
	} else {
		buttons = [][]button.InlineKeyboardButton{
			{{Text: "« ʙᴀᴄᴋ", CallbackData: "cmd:about", Style: "primary"}, button.CloseLocal()},
		}
	}

	return &message.Message{
		Text:     text,
		Keyboard: buttons,
	}
}

func (c *Config) GetPrivacyMessage() *message.Message {
	var (
		text    string
		buttons [][]button.InlineKeyboardButton
	)

	if c.PrivacyText != "" {
		text = c.PrivacyText
	} else {
		text = `
<blockquote expandable><b>Privacy Policy 📜</b>
<i>This bot stores the <b>publicly</b> visible data of users that is <b>required</b> for the bot to operate.

The following data of a user could be saved:
‣ Id
‣ Name
‣ Username
‣ Join Requests

ℹ️ Use the /uinfo command with your user id to view data stored about you.</i>
</blockquote>
`
	}

	if len(c.PrivacyButtons) != 0 {
		buttons = c.PrivacyButtons
	} else {
		buttons = [][]button.InlineKeyboardButton{
			{{Text: "« ʙᴀᴄᴋ", CallbackData: "cmd:help", Style: "primary"}, button.CloseLocal()},
		}
	}

	return &message.Message{
		Text:     text,
		Keyboard: buttons,
	}
}
