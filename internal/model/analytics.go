package model

import "time"

// SearchStat tracks the frequency of search queries.
type SearchStat struct {
	Query string `bson:"_id"`
	Count int64  `bson:"count"`
}

// AnalyticsResult holds the aggregated statistics for the admin dashboard.
type AnalyticsResult struct {
	TotalUsers    int64
	NewUsersToday int64
	ActiveUsers   int64 // Last 24h
	TopSearches   []SearchStat
	Languages     map[string]int64
	Countries     map[string]int64
	CachedAt      time.Time
}
