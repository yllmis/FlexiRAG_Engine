package ratelimit

import (
	"sync"
	"time"

	"flexirag-engine/internal/core"

	"golang.org/x/time/rate"
)

var _ core.RateLimiter = (*InMemoryRateLimiter)(nil)

type InMemoryRateLimiter struct {
	mu      sync.RWMutex             // 使用读写锁，提升性能
	limit   rate.Limit               // 补充速率 (每秒多少个)
	burst   int                      // 桶的容量 (允许的最大突发)
	buckets map[string]*rate.Limiter // 存储每个 Key 的官方限流器
	nowFn   func() time.Time
}

func NewInMemoryRateLimiter(limitPerMinute int) *InMemoryRateLimiter {
	if limitPerMinute <= 0 {
		limitPerMinute = 60
	}
	limitPerSec := rate.Limit(float64(limitPerMinute) / 60.0)
	return &InMemoryRateLimiter{
		limit:   limitPerSec,
		burst:   limitPerMinute,
		buckets: make(map[string]*rate.Limiter),
		nowFn:   time.Now,
	}
}

func (l *InMemoryRateLimiter) Allow(key string) bool {
	b := l.getBucket(key)
	return b.AllowN(l.nowFn(), 1)
}

func newInMemoryRateLimiterWithNow(limitPerMinute int, nowFn func() time.Time) *InMemoryRateLimiter {
	l := NewInMemoryRateLimiter(limitPerMinute)
	if nowFn != nil {
		l.nowFn = nowFn
	}
	return l
}

func (l *InMemoryRateLimiter) getBucket(key string) *rate.Limiter {
	l.mu.RLock()
	b, ok := l.buckets[key]
	l.mu.RUnlock()
	if ok {
		return b
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check，避免并发场景下重复创建。
	b, ok = l.buckets[key]
	if ok {
		return b
	}
	b = rate.NewLimiter(l.limit, l.burst)
	l.buckets[key] = b
	return b
}
