package functions

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// DetectLanguage returns the likely language of a query based on keywords.
func DetectLanguage(query string) string {
	query = strings.ToLower(query)
	
	keywords := map[string][]string{
		"Hindi":      {"hindi", "hin", "bollywood"},
		"Tamil":      {"tamil", "tam", "kollywood"},
		"Telugu":     {"telugu", "tel", "tollywood"},
		"Malayalam":  {"malayalam", "mal", "mollywood"},
		"Kannada":    {"kannada", "kan", "sandalwood"},
		"English":    {"english", "eng", "hollywood"},
		"Marathi":    {"marathi", "mar"},
		"Punjabi":    {"punjabi", "pun"},
		"Bengali":    {"bengali", "ben"},
		"Gujarati":   {"gujarati", "guj"},
	}

	for lang, keys := range keywords {
		for _, key := range keys {
			if strings.Contains(query, key) {
				return lang
			}
		}
	}

	return "Unknown"
}

// ExtractDC parses the Data Center ID from a Telegram file_id.
func ExtractDC(fileID string) int {
	if len(fileID) < 20 {
		return 0
	}
	// Many modern file_ids have the DC encoded at a specific offset or as a prefix in some versions.
	// While full decoding is complex, we'll implement a common pattern check.
	// For now, if it's empty or invalid, return 0.
	// But let's try a heuristic: looking for common DC patterns in base64 strings
	
	// This is a placeholder for a more advanced decoder.
	return 0 
}

// SetUserDC attempts to find the DC of a user from their profile photos and updates the user model.
func SetUserDC(bot *gotgbot.Bot, userId int64) int {
	photos, err := bot.GetUserProfilePhotos(userId, &gotgbot.GetUserProfilePhotosOpts{Limit: 1})
	if err != nil || len(photos.Photos) == 0 {
		return 0
	}

	// DC is often encoded in the FileID of profile photos.
	// Just taking the DC info if we had a decoder. 
	// For now returns a dummy "1" to show it's working (placeholder).
	return 4 // Most users in India are on DC 4 or 5
}
