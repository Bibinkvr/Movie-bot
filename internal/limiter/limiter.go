package limiter

import (
	"time"
)

var globalLimiter *tickerLimiter

type tickerLimiter struct {
	ticker *time.Ticker
}

func init() {
	// Default to 12 req/sec
	Initialize(12)
}

// Initialize sets up the global rate limiter.
func Initialize(reqPerSec int) {
	if globalLimiter != nil {
		globalLimiter.ticker.Stop()
	}
	interval := time.Second / time.Duration(reqPerSec)
	globalLimiter = &tickerLimiter{
		ticker: time.NewTicker(interval),
	}
}

// Wait blocks until the next slot is available.
func Wait() {
	if globalLimiter != nil {
		<-globalLimiter.ticker.C
	}
}
