package mongo

import (
	"autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SaveBroadcast saves a broadcast to the database.
func (c *Client) SaveBroadcast(b *model.Broadcast) error {
	_, err := c.broadcastCollection.InsertOne(c.ctx, b)
	return err
}

// GetBroadcast fetches a broadcast by its ID.
func (c *Client) GetBroadcast(id string) (*model.Broadcast, error) {
	var b model.Broadcast
	err := c.broadcastCollection.FindOne(c.ctx, bson.M{"_id": id}).Decode(&b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// UpdateBroadcast updates a broadcast document.
func (c *Client) UpdateBroadcast(id string, updates map[string]interface{}) error {
	_, err := c.broadcastCollection.UpdateOne(c.ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

// GetAllBroadcasts fetches all broadcasts, ordered by creation time descending.
func (c *Client) GetAllBroadcasts() ([]model.Broadcast, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := c.broadcastCollection.Find(c.ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(c.ctx)

	var list []model.Broadcast
	for cursor.Next(c.ctx) {
		var b model.Broadcast
		if err := cursor.Decode(&b); err == nil {
			list = append(list, b)
		}
	}
	return list, nil
}

// DeleteBroadcast deletes a broadcast document.
func (c *Client) DeleteBroadcast(id string) error {
	_, err := c.broadcastCollection.DeleteOne(c.ctx, bson.M{"_id": id})
	return err
}
