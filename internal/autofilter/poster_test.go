package autofilter

import "testing"

func TestTitleMatchesQuery(t *testing.T) {
	tests := []struct {
		title string
		query string
		want  bool
	}{
		{"Spider-Man: No Way Home", "spiderman no way home 2021", true},
		{"Spider-Man: No Way Home", "spider-man no way home 2021", true},
		{"Spiderman: No Way Home", "spiderman no way home 2021", true},
		{"Spiderman: No Way Home", "spiderman", true},
		{"The Amazing Spiderman", "amazing spiderman 2012", true},
		{"The Amazing Spider-Man 2", "amazing spiderman 2 2014", true},
		{"Spider-Man", "spiderman 1", true},
		{"Spider-Man 2", "spiderman 2", true},
		{"Spider-Man", "spiderman 2", false}, // Spider-Man 2 query should not match Spider-Man 1 poster
	}

	for _, tt := range tests {
		got := titleMatchesQuery(tt.title, tt.query)
		if got != tt.want {
			t.Errorf("titleMatchesQuery(%q, %q) = %v, want %v", tt.title, tt.query, got, tt.want)
		}
	}
}
