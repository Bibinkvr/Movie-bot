/*
Package autofilter contains types and methods to help work with autofilter results.
*/
package autofilter

import (
	"context"

	"autofilterbot/internal/database"
	"autofilterbot/internal/model"
)

type FilesFromCursorOptions interface {
	GetMaxResults() int
	GetMaxPages() int
	GetMaxPerPage() int
}

// FilesFromCursor loops through a cursor and outputs an array of files.
func FilesFromCursor(ctx context.Context, c database.Cursor, opts FilesFromCursorOptions) ([]Files, error) {
	var (
		totalCount int
		finished   bool
		totalFiles = make([]Files, 0, opts.GetMaxResults())
	)

	for i := 0; i < opts.GetMaxPages(); i++ {
		row := make([]File, 0, opts.GetMaxPerPage())

		for len(row) < opts.GetMaxPerPage() {
			if !c.Next(ctx) {
				finished = true
				break
			}

			var f model.File

			err := c.Decode(&f)
			if err != nil {
				return totalFiles, err
			}

			// Filter out samples, trailers, srt, etc. early
			if IsGarbageFile(f.FileName) {
				continue
			}

			row = append(row, File{File: f})
			totalCount++
			if totalCount >= opts.GetMaxResults() {
				finished = true
				break
			}
		}

		if len(row) != 0 {
			totalFiles = append(totalFiles, row)
		}

		if finished {
			return totalFiles, nil
		}
	}

	return totalFiles, nil
}
