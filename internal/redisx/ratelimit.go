package redisx

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RateLimiter 用 Redis 做 API 限流（固定窗口算法）。
//
// 场景：防止有人疯狂调 POST /api/chat，把 DeepSeek 额度刷光。
//
// 原理（小白版）：
//   - 每个 IP 每分钟一个计数器，存在 Redis 里
//   - 每来一次请求 INCR +1
//   - 超过 REDIS_RATE_LIMIT（如 60）就拒绝，返回 429
//
// Redis Key 示例：
//
//	ratelimit:chat:127.0.0.1:29381234
//	                      ↑ IP    ↑ 当前分钟编号（Unix时间/60）
type RateLimiter struct {
	rdb   *goredis.Client
	limit int // 每窗口最大请求数，来自 REDIS_RATE_LIMIT
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(rdb *goredis.Client, limit int) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit}
}

// Allow 判断本次请求是否放行。
//
// 返回：
//   - true  → 允许
//   - false → 超限，应返回 HTTP 429
func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// 未配置限流时直接放行
	if l == nil || l.rdb == nil || l.limit <= 0 {
		return true, nil
	}

	// 当前是第几个「分钟窗口」（从 1970 年算起的分钟数）
	window := time.Now().Unix() / 60
	redisKey := fmt.Sprintf("ratelimit:chat:%s:%d", key, window)

	// INCR：把计数器 +1，如果 key 不存在会从 0 变成 1
	count, err := l.rdb.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}

	// 第一次访问这个窗口时，设置 1 分钟过期（自动清理旧 key）
	if count == 1 {
		_ = l.rdb.Expire(ctx, redisKey, time.Minute).Err()
	}

	return count <= int64(l.limit), nil
}
