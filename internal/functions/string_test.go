package functions

import "testing"

func TestRemoveSymbols(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Cousins_and_Kalyanams_S01_E01", "Cousins and Kalyanams S01 E01"},
		{"Hello.World.2026.720p", "Hello World 2026 720p"},
		{"[TG] Movie @Channel_Name (2025)", "TG Movie Channel Name 2025"},
		{"Test---name+++with---symbols", "Test name with symbols"},
	}

	for _, test := range tests {
		result := RemoveSymbols(test.input)
		if result != test.expected {
			t.Errorf("RemoveSymbols(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestRemoveExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"movie.mkv", "movie"},
		{"series.s01e01.mp4", "series.s01e01"},
		{"no_extension", "no_extension"},
		{"archive.tar.gz", "archive.tar"}, // Last extension is .gz, which is cut
	}

	for _, test := range tests {
		result := RemoveExtension(test.input)
		if result != test.expected {
			t.Errorf("RemoveExtension(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}
