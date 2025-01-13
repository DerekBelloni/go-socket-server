package rateLimiter

import (
	"sync"
	"time"
)

type RateLimiter struct {
	ticker *time.Ticker
	Limit  int
	mu     sync.Mutex
	count  int
}

func NewRateLimiter(interval time.Duration, limit int) *RateLimiter {
	return &RateLimiter{
		ticker: time.NewTicker(interval),
		Limit:  limit,
		count:  0,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.count < rl.Limit {
		rl.count++
		return true
	}

	<-rl.ticker.C
	rl.count = 1
	return true
}
