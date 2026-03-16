package ratelimit

import (
	"testing"
	"time"
)

func TestInMemoryRateLimiter_AllowBurstAndIsolateByKey(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	limiter := newInMemoryRateLimiterWithNow(2, func() time.Time { return now })

	if !limiter.Allow("u1") {
		t.Fatal("第一次请求应放行")
	}
	if !limiter.Allow("u1") {
		t.Fatal("第二次请求应放行")
	}
	if limiter.Allow("u1") {
		t.Fatal("第三次请求应被限流")
	}
	if !limiter.Allow("u2") {
		t.Fatal("不同 key 应独立令牌桶")
	}
}

func TestInMemoryRateLimiter_RefillTokensOverTime(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	limiter := newInMemoryRateLimiterWithNow(2, func() time.Time { return now })

	if !limiter.Allow("u1") || !limiter.Allow("u1") {
		t.Fatal("初始突发容量应允许 2 次")
	}
	if limiter.Allow("u1") {
		t.Fatal("令牌用尽后应限流")
	}

	now = now.Add(30 * time.Second)
	if !limiter.Allow("u1") {
		t.Fatal("30 秒后应补充 1 个令牌")
	}
	if limiter.Allow("u1") {
		t.Fatal("仅补充 1 个令牌，下一次应限流")
	}

	now = now.Add(60 * time.Second)
	if !limiter.Allow("u1") {
		t.Fatal("继续补充后应再次放行")
	}
}
