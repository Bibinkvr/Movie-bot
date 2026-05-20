package mongo

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"autofilterbot/internal/database"
	"autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (c *Client) SaveFile(f *model.File) error {
	// Find any with matching file_id
	if res := c.fileCollection.FindOne(c.ctx, fileIdFilter(f.FileId)); res.Err() != mongo.ErrNoDocuments {
		return database.FileAlreadyExistsError{FileName: f.FileName}
	}

	// Find a document that starts with the same file_name and is within a 100 byte range of file_size
	duplicateFilter := bson.D{
		{Key: "file_name", Value: bson.D{{Key: "$regex", Value: "^" + f.FileName}}},
		{Key: "file_size", Value: bson.D{
			{Key: "$gte", Value: f.FileSize - 100},
			{Key: "$lte", Value: f.FileSize + 100},
		}},
	}
	if res := c.fileCollection.FindOne(c.ctx, duplicateFilter); res.Err() != mongo.ErrNoDocuments {
		return database.FileAlreadyExistsError{FileName: f.FileName}
	}

	_, err := c.fileCollection.InsertOne(c.ctx, f)

	return err
}

func (c *Client) SaveFiles(files ...*model.File) []error {
	var errs []error

	for _, f := range files {
		if err := c.SaveFile(f); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c *Client) BulkSaveFiles(files []*model.File) error {
	if len(files) == 0 {
		return nil
	}

	var models []mongo.WriteModel
	for _, f := range files {
		filter := bson.D{{Key: "file_id", Value: f.FileId}}
		upsert := true
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.D{{Key: "$setOnInsert", Value: f}}).
			SetUpsert(upsert))
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := c.fileCollection.BulkWrite(c.ctx, models, opts)
	return err
}

func (c *Client) GetFile(fileId string) (*model.File, error) {
	res := c.fileCollection.FindOne(c.ctx, idFilter(fileId))
	if err := res.Err(); err != nil {
		return nil, err
	}

	var f model.File

	err := res.Decode(&f)

	return &f, err
}

func (c *Client) DeleteFile(fileId string) error {
	_, err := c.fileCollection.DeleteOne(c.ctx, idFilter(fileId))
	return err
}

func (c *Client) SearchFiles(query string) (database.Cursor, error) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, mongo.ErrNilDocument
	}

	pattern := "(?i)"
	for _, word := range words {
		// Use word boundaries (non-alphanumeric or start/end) to prevent partial matches like 'LOKiHD' matching 'Loki'
		quotedWord := regexp.QuoteMeta(word)
		pattern += fmt.Sprintf("(?=.*(?:[^a-zA-Z0-9]|^)%s(?:[^a-zA-Z0-9]|$))", quotedWord)
	}
	searchTerms := strings.Join(words, " ")

	pipeline := bson.D{
		{Key: "$text", Value: bson.D{{Key: "$search", Value: searchTerms}}},
		{Key: "file_name", Value: bson.D{{Key: "$regex", Value: pattern}}},
	}

	return c.fileCollection.Find(context.Background(), pipeline, options.Find().SetSort(bson.M{"time": -1}).SetLimit(50))
}

// fileIdFilter creates a bson filter to match by file_id.
func fileIdFilter(id string) bson.D {
	return bson.D{{Key: "file_id", Value: id}}
}
