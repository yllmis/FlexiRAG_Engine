package ratelimit

import (
	"sync"
	"time"

	"flexirag-engine/internal/core"
)

var _ core.RateLimiter = (*InMemoryRateLimiter)(nil)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type InMemoryRateLimiter struct {
	mu           sync.Mutex
	capacity     float64
	refillPerSec float64
	buckets      map[string]*bucket
	nowFn        func() time.Time
}

func NewInMemoryRateLimiter(limitPerMinute int) *InMemoryRateLimiter {
	if limitPerMinute <= 0 {
		limitPerMinute = 60
	}
	refillPerSec := float64(limitPerMinute) / 60.0
	return &InMemoryRateLimiter{
		capacity:     float64(limitPerMinute),
		refillPerSec: refillPerSec,
		buckets:      make(map[string]*bucket),
		nowFn:        time.Now,
	}
}

func (l *InMemoryRateLimiter) Allow(key string) bool {
	now := l.nowFn()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.capacity - 1, lastRefill: now}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.refillPerSec
		if b.tokens > l.capacity {
			b.tokens = l.capacity
		}
		b.lastRefill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func newInMemoryRateLimiterWithNow(limitPerMinute int, nowFn func() time.Time) *InMemoryRateLimiter {
	l := NewInMemoryRateLimiter(limitPerMinute)
	if nowFn != nil {
		l.nowFn = nowFn
	}
	return l
}
