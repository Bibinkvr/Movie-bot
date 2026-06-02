package mongo

import (
	"time"

	"autofilterbot/internal/database"
	"autofilterbot/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TrackSearch increments the frequency of a search query.
func (c *Client) TrackSearch(query string) error {
	coll := c.db.Collection("SearchStats")
	filter := bson.M{"_id": query}
	update := bson.M{"$inc": bson.M{"count": 1}}
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(c.ctx, filter, update, opts)
	return err
}

// GetTopSearches returns the most searched queries.
func (c *Client) GetTopSearches(limit int) ([]model.SearchStat, error) {
	coll := c.db.Collection("SearchStats")
	opts := options.Find().SetSort(bson.M{"count": -1}).SetLimit(int64(limit))
	cursor, err := coll.Find(c.ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var results []model.SearchStat
	err = cursor.All(c.ctx, &results)
	return results, err
}

// UpdateUserLastSeen updates the last_search_at field.
func (c *Client) UpdateUserLastSeen(userId int64) error {
	filter := idFilter(userId)
	update := bson.M{"$set": bson.M{"last_search_at": time.Now().Unix()}}
	_, err := c.userCollection.UpdateOne(c.ctx, filter, update)
	return err
}

// GetUserAnalytics performs aggregations for the stats dashboard.
func (c *Client) GetUserAnalytics() (total, newToday, active24h, activeWeekly, activeMonthly int64, countries map[string]int64, err error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	twentyFourHoursAgo := now.Add(-24 * time.Hour).Unix()
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour).Unix()
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour).Unix()

	// Total Users
	total, err = c.userCollection.CountDocuments(c.ctx, bson.M{})
	if err != nil {
		return
	}

	// New Today
	newToday, err = c.userCollection.CountDocuments(c.ctx, bson.M{"created_at": bson.M{"$gte": todayStart}})
	if err != nil {
		return
	}

	// Active 24h
	active24h, err = c.userCollection.CountDocuments(c.ctx, bson.M{"last_search_at": bson.M{"$gte": twentyFourHoursAgo}})
	if err != nil {
		return
	}

	// Active Weekly
	activeWeekly, err = c.userCollection.CountDocuments(c.ctx, bson.M{"last_search_at": bson.M{"$gte": sevenDaysAgo}})
	if err != nil {
		return
	}

	// Active Monthly
	activeMonthly, err = c.userCollection.CountDocuments(c.ctx, bson.M{"last_search_at": bson.M{"$gte": thirtyDaysAgo}})
	if err != nil {
		return
	}

	// Country Distribution
	countries = make(map[string]int64)
	pipeline := []bson.M{
		{"$group": bson.M{"_id": bson.M{"$ifNull": []interface{}{"$country", "Other"}}, "count": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"count": -1}},
	}
	cursor, err := c.userCollection.Aggregate(c.ctx, pipeline)
	if err == nil {
		var results []struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err = cursor.All(c.ctx, &results); err == nil {
			for _, r := range results {
				name := r.ID
				if name == "" {
					name = "Other"
				}
				countries[name] = r.Count
			}
		}
	}

	return
}
// GetFsubAnalytics fetches detailed metrics for a specific fsub channel.
func (c *Client) GetFsubAnalytics(channelID int64) (model.FsubStats, error) {
	var s model.FsubStats

	// 1. Total Requests in Channel (all users who sent request)
	count, err := c.joinRequestsCollection.CountDocuments(c.ctx, bson.M{"join_requests": channelID})
	if err == nil {
		s.TotalRequests = count
	}

	// 2. Bot Users who requested (in both JoinRequests and Users collection)
	pipeline := []bson.M{
		{"$match": bson.M{"join_requests": channelID}},
		{"$lookup": bson.M{
			"from":         database.CollectionNameUsers,
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "user_info",
		}},
		{"$unwind": "$user_info"},
		{"$count": "count"},
	}
	cursor, err := c.joinRequestsCollection.Aggregate(c.ctx, pipeline)
	if err == nil {
		var results []struct {
			Count int64 `bson:"count"`
		}
		if err = cursor.All(c.ctx, &results); err == nil && len(results) > 0 {
			s.Requested = results[0].Count
		}
	}

	// 3. Bot Users Count
	s.BotUsers, _ = c.userCollection.CountDocuments(c.ctx, bson.M{})

	// 4. Daily, Weekly, and Monthly stats
	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour).Unix()
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour).Unix()
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour).Unix()

	s.DailyRequests, _ = c.joinRequestsLogsCollection.CountDocuments(c.ctx, bson.M{
		"channel_id":   channelID,
		"requested_at": bson.M{"$gte": twentyFourHoursAgo},
	})
	s.WeeklyRequests, _ = c.joinRequestsLogsCollection.CountDocuments(c.ctx, bson.M{
		"channel_id":   channelID,
		"requested_at": bson.M{"$gte": sevenDaysAgo},
	})
	s.MonthlyRequests, _ = c.joinRequestsLogsCollection.CountDocuments(c.ctx, bson.M{
		"channel_id":   channelID,
		"requested_at": bson.M{"$gte": thirtyDaysAgo},
	})

	return s, nil
}
