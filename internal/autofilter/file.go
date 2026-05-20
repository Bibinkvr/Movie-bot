package autofilter

import (
	"fmt"

	"autofilterbot/internal/functions"
	"autofilterbot/internal/model"
	"autofilterbot/pkg/shortener"
	"slices"
	"strings"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Files []File

// File wraps model.File with some search result specific data.
type File struct {
	model.File
	// Indicates whether the file was selected from the selection menu.
	IsSelected bool `json:"selected,omitempty"`
}

// Process returns a slice of buttons to be used in message markup.
func (files Files) Process(chatId int64, botUsername string, opts ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	return ProcessFiles(files, chatId, botUsername, opts)
}

type ProcessFilesOptions interface {
	GetButtonTemplate() string
	GetSizeButton() bool
	GetShortener() shortener.Shortener
}

// ProcessFiles changes files into a keyboard slice to be used as markup in a message.
func ProcessFiles(files Files, chatId int64, botUsername string, opts ProcessFilesOptions) [][]gotgbot.InlineKeyboardButton {
	var (
		hasShortener = opts.GetShortener().ApiKey != ""
		result       = make([][]gotgbot.InlineKeyboardButton, 0, len(files))
		isMovie      = DetectType(files) == "movie"
	)

	for _, f := range files {
		// Filter out samples, trailers, srt, etc.
		if IsGarbageFile(f.FileName) {
			continue
		}

		size := functions.FileSizeToString(f.FileSize)

		var (
			btnText string
			btn     gotgbot.InlineKeyboardButton
		)

		if isMovie {
			// Smart naming for movies
			label := ExtractMovieLabel(f.FileName)
			if label == "" {
				label = f.FileName
			}
			btnText = fmt.Sprintf("🔹 %s 🍿 %s", size, label)
		} else {
			btnText = fmt.Sprintf("🔹 %s 🍿 %s", size, f.FileName)
		}

		if chatId > 0 {
			// Private chat: Send directly via callback
			btn = gotgbot.InlineKeyboardButton{Text: btnText, CallbackData: "sendfile|" + f.UniqueId, Style: "primary"}
		} else {
			// Group or unknown: Use start URL
			url := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, URLData{
				FileUniqueId: f.UniqueId,
				ChatId:       chatId,
				HasShortener: hasShortener,
			}.Encode())
			btn = gotgbot.InlineKeyboardButton{Text: btnText, Url: url, Style: "primary"}
		}

		result = append(result, []gotgbot.InlineKeyboardButton{btn})
	}

	return result
}

// SortSeries sorts files by Season (ASC), Episode (ASC), and Quality (DESC).
func (files Files) SortSeries() {
	slices.SortFunc(files, func(a, b File) int {
		s1, e1 := ExtractSeriesMetadata(a.FileName)
		s2, e2 := ExtractSeriesMetadata(b.FileName)
		
		if s1 != s2 {
			return s1 - s2
		}
		if e1 != e2 {
			return e1 - e2
		}
		
		q1 := QualityLevel(a.FileName)
		q2 := QualityLevel(b.FileName)
		return q2 - q1 // Quality DESC
	})
}

// SortMovies sorts files by Quality (DESC), Size (DESC), and Name (ASC).
func (files Files) SortMovies() {
	slices.SortFunc(files, func(a, b File) int {
		q1 := QualityLevel(a.FileName)
		q2 := QualityLevel(b.FileName)
		if q1 != q2 {
			return q2 - q1 // Quality DESC
		}
		
		if a.FileSize != b.FileSize {
			if b.FileSize > a.FileSize {
				return 1
			}
			return -1
		}
		
		return 0
	})
}

// FilterBySeason returns only files belonging to a specific season.
func (files Files) FilterBySeason(s int) Files {
	result := make(Files, 0, len(files))
	for _, f := range files {
		fs, _ := ExtractSeriesMetadata(f.FileName)
		if fs == s {
			result = append(result, f)
		}
	}
	return result
}

// FilterByLanguage returns only files containing the specified language tag.
func (files Files) FilterByLanguage(lang string) Files {
	result := make(Files, 0, len(files))
	lang = strings.ToLower(lang)
	
	patterns := map[string][]string{
		"hindi":     {"hindi", "hin"},
		"english":   {"english", "eng"},
		"tamil":     {"tamil", "tam"},
		"telugu":    {"telugu", "tel"},
		"malayalam": {"malayalam", "mal"},
		"kannada":   {"kannada", "kan"},
		"multi":     {"multi", "dual", "mux"},
	}

	searchTags, ok := patterns[lang]
	if !ok {
		// If not in our patterns, just try literal match
		searchTags = []string{lang}
	}

	for _, f := range files {
		lower := strings.ToLower(f.FileName)
		found := false
		for _, tag := range searchTags {
			if strings.Contains(lower, tag) {
				found = true
				break
			}
		}
		if found {
			result = append(result, f)
		}
	}
	return result
}

// ProcessSeasons returns a keyboard with season buttons for series.
func (files Files) ProcessSeasons(uniqueId string) [][]gotgbot.InlineKeyboardButton {
	groups := GroupBySeason(files)
	keyboard := make([][]gotgbot.InlineKeyboardButton, 0, (len(groups)/2)+1)
	
	// Sort seasons for consistent UI
	seasons := make([]int, 0, len(groups))
	for s := range groups {
		seasons = append(seasons, s)
	}
	slices.Sort(seasons)

	var currentRow []gotgbot.InlineKeyboardButton
	for _, s := range seasons {
		if s == 0 {
			continue // Skip 'Other' button for cleaner UI
		}
		text := fmt.Sprintf("💠 S%d", s)
		currentRow = append(currentRow, gotgbot.InlineKeyboardButton{
			Text:         text,
			CallbackData: fmt.Sprintf("sn|%s_%d", uniqueId, s),
			Style:        "primary",
		})
		if len(currentRow) == 2 {
			keyboard = append(keyboard, currentRow)
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		keyboard = append(keyboard, currentRow)
	}

	return keyboard
}

// SelectMenu returns a keyboard with to select files from.
func (files Files) SelectMenu(uniqueId string, pageIndex int) [][]gotgbot.InlineKeyboardButton {
	keyboard := make([][]gotgbot.InlineKeyboardButton, 0, len(files))

	for _, f := range files {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{
			Text:         fmt.Sprintf("%s[%s] %s", tick(f.IsSelected), functions.FileSizeToString(f.FileSize), f.FileName),
			CallbackData: fmt.Sprintf("sel|%s_%d_%s", uniqueId, pageIndex, f.UniqueId),
		}})
	}

	return keyboard
}

// tick returns a tick symbol if val is true.
func tick(val bool) string {
	if val {
		return "✅ "
	}

	return ""
}
