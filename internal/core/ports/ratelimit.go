package ports

// RateLimiter 定义限流器标准
type RateLimiter interface {
	Allow(key string) bool
}