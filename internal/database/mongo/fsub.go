package mongo

import (
	"time"

	"autofilterbot/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var upsert = true

func (c *Client) SaveUserJoinRequest(userId, chatId int64) error {
	_, err := c.joinRequestsCollection.UpdateOne(
		c.ctx,
		idFilter(userId),
		bson.D{{Key: "$addToSet", Value: bson.D{{Key: "join_requests", Value: chatId}}}},
		&options.UpdateOptions{Upsert: &upsert},
	)
	if err != nil {
		return err
	}

	// Log the join request timestamp
	filter := bson.M{"user_id": userId, "channel_id": chatId}
	update := bson.M{
		"$setOnInsert": bson.M{
			"requested_at": time.Now().Unix(),
		},
	}
	_, err = c.joinRequestsLogsCollection.UpdateOne(c.ctx, filter, update, &options.UpdateOptions{Upsert: &upsert})
	return err
}

func (c *Client) DeleteUserJoinRequest(userId, chatId int64) error {
	_, err := c.joinRequestsCollection.UpdateOne(
		c.ctx,
		idFilter(userId),
		bson.D{{Key: "$pull", Value: bson.D{{Key: "join_requests", Value: chatId}}}},
		&options.UpdateOptions{Upsert: &upsert},
	)
	if err != nil {
		return err
	}

	_, err = c.joinRequestsLogsCollection.DeleteOne(c.ctx, bson.M{"user_id": userId, "channel_id": chatId})
	return err
}

// GetUser fetches a user's join requests from the database by id.
func (c *Client) GetUserJoinRequests(userId int64) (*model.User, error) {
	var u model.User

	res := c.joinRequestsCollection.FindOne(c.ctx, idFilter(userId))
	if err := res.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return &model.User{UserId: userId}, nil
		}
		return nil, err
	}

	err := res.Decode(&u)

	return &u, err
}
