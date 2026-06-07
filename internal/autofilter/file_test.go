package autofilter

import (
	"testing"

	"autofilterbot/internal/model"
)

func TestFormatFileButtonText(t *testing.T) {
	// 1. Movie Test
	movieResult := FormatFileButtonText("Dacoit Ek Prem Katha 2026 Hindi 720p.mkv", 1011823084)
	expectedMovie := "[𝟵𝟲𝟰.𝟵𝟱 𝗠𝗕] ➤ 𝗗𝗮𝗰𝗼𝗶𝘁 𝗘𝗸 𝗣𝗿𝗲𝗺 𝗞𝗮𝘁𝗵𝗮 (2026)  𝗛𝗶𝗻𝗱𝗶 • 𝟳𝟮𝟬𝗣"
	if movieResult != expectedMovie {
		t.Errorf("\nExpected movie result:\n%s\nGot:\n%s", expectedMovie, movieResult)
	}

	// 2. Series Test
	seriesResult := FormatFileButtonText("[S01E01] Dark Hole 2026 English 1080p.mkv", 472078336)
	expectedSeries := "[𝟰𝟱𝟬.𝟮𝟭 𝗠𝗕] ➤ [𝗦𝟬𝟭𝗘𝟬𝟭] 𝗗𝗮𝗿𝗸 𝗛𝗼𝗹𝗲 (2026)  𝗘𝗻𝗴𝗹𝗶𝘀𝗵 • 𝟭𝟬𝟴𝟬𝗣"
	if seriesResult != expectedSeries {
		t.Errorf("\nExpected series result:\n%s\nGot:\n%s", expectedSeries, seriesResult)
	}
}

func TestFilterByLanguage(t *testing.T) {
	files := Files{
		{File: model.File{FileName: "O.K.Kanmani.2015.Malayalam.mkv"}},
		{File: model.File{FileName: "KGF.Chapter.2.2022.Kannada.mkv"}},
		{File: model.File{FileName: "Movie.With.Kan.Subtitles.mkv"}},
	}

	// 1. Kannada query should not match Malayalam "Kanmani", but should match "Kannada" and "Kan" (since "Kan" is a word)
	kannadaFiltered := files.FilterByLanguage("kannada")
	if len(kannadaFiltered) != 2 {
		t.Errorf("Expected 2 Kannada files, got %d", len(kannadaFiltered))
	}
	for _, f := range kannadaFiltered {
		if f.FileName == "O.K.Kanmani.2015.Malayalam.mkv" {
			t.Errorf("Should not match Kannada to O.K.Kanmani.2015.Malayalam.mkv")
		}
	}

	// 2. Malayalam query should match "O.K.Kanmani.2015.Malayalam.mkv"
	malayalamFiltered := files.FilterByLanguage("malayalam")
	if len(malayalamFiltered) != 1 {
		t.Errorf("Expected 1 Malayalam file, got %d", len(malayalamFiltered))
	}
	if malayalamFiltered[0].FileName != "O.K.Kanmani.2015.Malayalam.mkv" {
		t.Errorf("Expected Malayalam file to match O.K.Kanmani.2015.Malayalam.mkv")
	}
}
