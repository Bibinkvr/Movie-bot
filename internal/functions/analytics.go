package functions

import (
	"strings"
)

// DetectCountry maps a Telegram language code to a country name.
func DetectCountry(langCode string) string {
	switch strings.ToLower(langCode) {
	case "hi":
		return "India"
	case "ta":
		return "Tamil Nadu/India"
	case "ml":
		return "Kerala/India"
	case "te":
		return "Telugu/India"
	case "kn":
		return "Karnataka/India"
	case "en":
		return "Global/English"
	case "ar":
		return "Middle East/Arabic"
	case "ru":
		return "Russia"
	case "uz":
		return "Uzbekistan"
	case "id":
		return "Indonesia"
	case "es":
		return "Spanish/Global"
	default:
		return "Other"
	}
}
