package autofilter

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	seasonEpisodeRegex = regexp.MustCompile(`(?i)\bs(\d+)\s?e(\d+)\b|\bseason\s?(\d+)\s?episode\s?(\d+)\b|\b(\d+)x(\d+)\b`)
	qualityRegex       = regexp.MustCompile(`(?i)(\d{3,4}p|bluray|web-dl|webrip|hdtv|camrip|brrip|h264|h265|x264|x265)`)
	seasonOnlyRegex    = regexp.MustCompile(`(?i)\bseason\s?(\d+)\b|\bs(\d+)\b`)
)

type MovieMetadata struct {
	Quality    string
	Resolution string
}

type SeriesMetadata struct {
	Season  int
	Episode int
}

// DetectType returns "series" if at least one file follows a series pattern, else "movie".
func DetectType(files []File) string {
	for _, f := range files {
		if IsSeriesFile(f.FileName) {
			return "series"
		}
	}
	return "movie"
}

// IsSeriesFile returns true if the filename matches a season/episode pattern.
func IsSeriesFile(name string) bool {
	lower := strings.ToLower(name)
	return seasonEpisodeRegex.MatchString(lower) || seasonOnlyRegex.MatchString(lower)
}

// GroupBySeason groups files by their season number.
func GroupBySeason(files []File) map[int][]File {
	groups := make(map[int][]File)
	for _, f := range files {
		s, _ := ExtractSeriesMetadata(f.FileName)
		groups[s] = append(groups[s], f)
	}
	return groups
}

// ExtractSeriesMetadata parses season and episode from filename.
func ExtractSeriesMetadata(name string) (int, int) {
	lower := strings.ToLower(name)
	
	var s, e int
	matches := seasonEpisodeRegex.FindStringSubmatch(lower)
	
	if len(matches) > 0 {
		// s01e01 pattern (matches[1] and matches[2])
		if len(matches) > 2 && matches[1] != "" && matches[2] != "" {
			s, _ = strconv.Atoi(matches[1])
			e, _ = strconv.Atoi(matches[2])
		} else if len(matches) > 4 && matches[3] != "" && matches[4] != "" {
			// season 01 episode 01 pattern (matches[3] and matches[4])
			s, _ = strconv.Atoi(matches[3])
			e, _ = strconv.Atoi(matches[4])
		} else if len(matches) > 6 && matches[5] != "" && matches[6] != "" {
			// 01x01 pattern (matches[5] and matches[6])
			s, _ = strconv.Atoi(matches[5])
			e, _ = strconv.Atoi(matches[6])
		}
	}

	// Fallback to season only if no season was found or no episode was found
	if s == 0 {
		match := seasonOnlyRegex.FindStringSubmatch(lower)
		if len(match) > 1 && match[1] != "" {
			s, _ = strconv.Atoi(match[1])
		} else if len(match) > 2 && match[2] != "" {
			s, _ = strconv.Atoi(match[2])
		}
	}
	
	return s, e
}

// QualityLevel returns a numeric priority for quality tags.
func QualityLevel(name string) int {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "2160p") || strings.Contains(lower, "4k"):
		return 100
	case strings.Contains(lower, "1080p"):
		return 80
	case strings.Contains(lower, "bluray"):
		return 75
	case strings.Contains(lower, "720p"):
		return 60
	case strings.Contains(lower, "hdtv") || strings.Contains(lower, "web"):
		return 50
	case strings.Contains(lower, "480p"):
		return 40
	case strings.Contains(lower, "hdrip"):
		return 35
	case strings.Contains(lower, "webrip"):
		return 30
	case strings.Contains(lower, "dvdrip"):
		return 25
	case strings.Contains(lower, "hdtc") || strings.Contains(lower, "tc") || strings.Contains(lower, "telecine"):
		return 15
	case strings.Contains(lower, "camrip") || strings.Contains(lower, "cam") || strings.Contains(lower, "hdcam") || strings.Contains(lower, "ts"):
		return 10
	case strings.Contains(lower, "predvd") || strings.Contains(lower, "p-dvd"):
		return 5
	default:
		return 0
	}
}

// ExtractMovieLabel creates a clean label like "1080p BluRay"
func ExtractMovieLabel(name string) string {
	lower := strings.ToLower(name)
	
	// Find the first quality tag to determine where the movie name ends
	firstQualityIdx := -1
	
	match := qualityRegex.FindStringIndex(lower)
	if match != nil {
		firstQualityIdx = match[0]
	}

	var movieName string
	if firstQualityIdx != -1 {
		movieName = name[:firstQualityIdx]
	} else {
		movieName = name
	}

	// Clean movie name (remove dots, dashes, etc.)
	movieName = strings.ReplaceAll(movieName, ".", " ")
	movieName = strings.ReplaceAll(movieName, "-", " ")
	movieName = strings.TrimSpace(movieName)
	
	// Limit movie name length to keep buttons readable
	if len(movieName) > 35 {
		movieName = movieName[:32] + "..."
	}

	// Extract quality tags
	tags := qualityRegex.FindAllString(lower, -1)
	qualityStr := ""
	for _, t := range tags {
		qualityStr += strings.ToUpper(t) + " "
	}
	qualityStr = strings.TrimSpace(qualityStr)
	
	emoji := "🎬"
	if strings.Contains(lower, "2160p") || strings.Contains(lower, "4k") {
		emoji = "💎"
	} else if strings.Contains(lower, "1080p") {
		emoji = "✨"
	} else if strings.Contains(lower, "720p") {
		emoji = "⚡"
	}

	if movieName != "" && qualityStr != "" {
		return emoji + " " + movieName + " | " + qualityStr
	} else if movieName != "" {
		return emoji + " " + movieName
	}
	return qualityStr
}

// IsGarbageFile returns true for samples, trailers, etc.
func IsGarbageFile(name string) bool {
	lower := strings.ToLower(name)
	garbage := []string{"sample", "trailer", ".srt", ".txt", "nfo", "idx", "sub", "camrip", "predvd", "hdcam", "telecine", "hdtc", "p-dvd"}
	return slices.ContainsFunc(garbage, func(g string) bool {
		return strings.Contains(lower, g)
	})
}
// DetectLanguages extracts available languages from the file list.
func DetectLanguages(files []File) []string {
	langs := make(map[string]bool)
	patterns := map[string][]string{
		"Hindi":     {"hindi", "hin"},
		"English":   {"english", "eng"},
		"Tamil":     {"tamil", "tam"},
		"Telugu":    {"telugu", "tel"},
		"Malayalam": {"malayalam", "mal"},
		"Kannada":   {"kannada", "kan"},
		"Multi":     {"multi", "dual", "mux"},
	}

	for _, f := range files {
		lower := strings.ToLower(f.FileName)
		for name, p := range patterns {
			for _, s := range p {
				if strings.Contains(lower, s) {
					langs[name] = true
					break
				}
			}
		}
	}

	result := make([]string, 0, len(langs))
	for l := range langs {
		result = append(result, l)
	}
	slices.Sort(result)
	return result
}
