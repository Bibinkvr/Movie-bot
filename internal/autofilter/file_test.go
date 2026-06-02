package autofilter

import (
	"testing"
)

func TestFormatFileButtonText(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		fileSize int64
		expected string
	}{
		{
			name:     "Movie with brackets at start",
			fileName: "[Hindi] Spider-Man Noir (2024) 1080p BluRay 1200000000.mkv",
			fileSize: 1200000000,
			expected: "🎬 [1.12 GB] Spider Man Noir (2024) [Hindi • 1080P BLURAY]",
		},
		{
			name:     "Series SxxExx with leading brackets",
			fileName: "[Hindi] [Dual Audio] Ragnarok S01E06 720p WEB-DL 365760000.mkv",
			fileSize: 365760000,
			expected: "📺 [348.82 MB] Ragnarok S01E06 [Multi • 720P WEB-DL]",
		},
		{
			name:     "Series with Season only",
			fileName: "Ragnarok S01 720p 365760000.mkv",
			fileSize: 365760000,
			expected: "📺 [348.82 MB] Ragnarok S01 [720P]",
		},
		{
			name:     "Movie without year or language",
			fileName: "Spider-Man Noir 1080p.mkv",
			fileSize: 1200000000,
			expected: "🎬 [1.12 GB] Spider Man Noir [1080P]",
		},
		{
			name:     "Combined series episodes with range",
			fileName: "Cousins and Kalyanams S01 COMBINED E1 E4 1080p JHS WEB D.mkv",
			fileSize: 2523324416,
			expected: "📺 [2.35 GB] Cousins And Kalyanams S01 Combined E1-E4 [1080P]",
		},
		{
			name:     "Series episodes with range",
			fileName: "Ragnarok S01 E01-E04 720p.mkv",
			fileSize: 1200000000,
			expected: "📺 [1.12 GB] Ragnarok S01 E01-E04 [720P]",
		},
		{
			name:     "Series combined without range",
			fileName: "Ragnarok S01 COMBINED 720p.mkv",
			fileSize: 1200000000,
			expected: "📺 [1.12 GB] Ragnarok S01 Combined [720P]",
		},
		{
			name:     "Series Cousins with underscores and E01 E04 combined",
			fileName: "Cousins_and_Kalyanams_S01_E01_E04_COMBiNED_1080p_JHS_WEB_DLx264.mkv",
			fileSize: 2523971343,
			expected: "📺 [2.35 GB] Cousins And Kalyanams S01 E01-E04 [1080P]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatFileButtonText(tt.fileName, tt.fileSize)
			if actual != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, actual)
			}
		})
	}
}
