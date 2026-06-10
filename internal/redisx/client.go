// Package redisx 封装 Redis 连接（第 9 步扩展）。
//
// Redis 在本项目里做两件事：
//  1. 会话消息缓存（配合 MySQL，见 internal/store/cached）
//  2. HTTP /api/chat 限流（见 ratelimit.go）
//
// 我们使用官方 Go 客户端：github.com/redis/go-redis/v9
package redisx

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zhjing1019/goAiAgent/internal/config"
)

// Client 对 go-redis 客户端的薄封装。
//
// 为什么不直接在 app.go 里 new goredis.Client？
//   - 统一连接、Ping 验证、关闭逻辑
//   - 上层（cached、ratelimit）只关心「能用的 Redis 连接」
type Client struct {
	rdb *goredis.Client // 真正的 Redis 客户端
}

// Open 根据配置连接 Redis。
//
// 步骤：
//  1. goredis.NewClient 创建客户端（此时还没真正连上）
//  2. Ping 发一个心跳，确认 Redis 服务在跑
//  3. Ping 失败就 Close 并返回错误
func Open(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	// NewClient 只创建连接池对象，不会立刻建 TCP 连接
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,     // 例如 127.0.0.1:6379
		Password: cfg.Password, // 本地开发通常为空
		DB:       cfg.DB,       // Redis 有 16 个逻辑库 0~15，默认 0
	})

	// 用带超时的 context，避免 Redis 挂了一直卡住
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// RDB 暴露底层客户端，给 cached.Store 和 RateLimiter 使用。
func (c *Client) RDB() *goredis.Client {
	return c.rdb
}

// Close 程序退出时关闭连接池（在 app.Close 里调用）。
func (c *Client) Close() error {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}
