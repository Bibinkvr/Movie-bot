package core

import (
	"sync"
	"time"

	"autofilterbot/internal/model"
)

type AnalyticsService struct {
	cache      *model.AnalyticsResult
	cacheExpiry time.Time
	mu         sync.RWMutex
	app        interface{} // using interface to break import cycle
}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

func (s *AnalyticsService) SetApp(app interface{}) {
	s.app = app
}

func (s *AnalyticsService) GetStats() (*model.AnalyticsResult, error) {
	s.mu.RLock()
	if s.cache != nil && time.Now().Before(s.cacheExpiry) {
		res := *s.cache // Return a copy
		s.mu.RUnlock()
		return &res, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Re-check after acquiring lock
	if s.cache != nil && time.Now().Before(s.cacheExpiry) {
		return s.cache, nil
	}

	// Fetch from DB
	total, newToday, active24h, countries, err := _app.DB.GetUserAnalytics()
	if err != nil {
		return nil, err
	}

	topSearches, err := _app.DB.GetTopSearches(10)
	if err != nil {
		topSearches = []model.SearchStat{}
	}

	// Languages from Global Stats
	languages := make(map[string]int64)
	if stats, err := _app.DB.Stats(); err == nil {
		languages = stats.TopLanguages
	}

	s.cache = &model.AnalyticsResult{
		TotalUsers:    total,
		NewUsersToday: newToday,
		ActiveUsers:   active24h,
		TopSearches:   topSearches,
		Countries:     countries,
		Languages:     languages,
		CachedAt:      time.Now(),
	}
	s.cacheExpiry = time.Now().Add(60 * time.Second)

	return s.cache, nil
}
