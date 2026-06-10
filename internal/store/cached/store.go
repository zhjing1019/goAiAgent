// Package cached 用 Redis 缓存 MySQL 会话消息（Cache-Aside 旁路缓存）。
//
// ┌─────────┐    ① 先查 Redis     ┌───────┐
// │ Agent   │ ─────────────────▶ │ Redis │ 命中 → 直接返回，不查 MySQL
// │LoadSession                   └───────┘
// └────┬────┘    ② 未命中
//      │ ──────────────────────▶ MySQL 查到后 ③ 写入 Redis
//      ▼
//
// 为什么需要？
//   HTTP 是无状态的，每次 POST /api/chat 都要 LoadSession 读历史。
//   历史存在 MySQL，频繁 SELECT 压力大；Redis 内存读写快，适合做热数据缓存。
//
// 注意：MySQL 仍是「唯一真相来源」，Redis 只是加速层，丢了可以重建。
package cached

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"
)

// Store 装饰器模式：在 MySQL Store 外面包一层 Redis 缓存。
//
// 它实现了 store.SessionStore 接口，Agent 无感知——
// 以为在调 MySQL，其实可能走了 Redis。
type Store struct {
	underlying store.SessionStore // 真正的存储（MySQL）
	rdb        *goredis.Client  // Redis 客户端
	ttl        time.Duration    // 缓存过期时间，来自 REDIS_SESSION_TTL
	prefix     string           // Key 前缀，避免和其他项目冲突
}

// New 用 Redis 包装已有的 SessionStore。
//
// 调用时机：app.go 里 MySQL 和 Redis 都连上之后：
//
//	sessionStore = cached.New(mysqlStore, redisClient.RDB(), ttl)
func New(underlying store.SessionStore, rdb *goredis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Store{
		underlying: underlying,
		rdb:        rdb,
		ttl:        ttl,
		prefix:     "goagent:", // 所有 key 以 goagent: 开头，方便在 Redis 里辨认
	}
}

// msgKey 生成会话消息的 Redis Key。
//
// 示例：goagent:session:msgs:a1b2c3d4-...
func (s *Store) msgKey(sessionID string) string {
	return s.prefix + "session:msgs:" + sessionID
}

// Migrate 建表仍交给 MySQL 做。
func (s *Store) Migrate(ctx context.Context) error {
	return s.underlying.Migrate(ctx)
}

// CreateSession 创建会话仍写 MySQL（新会话还没有消息，无需缓存）。
func (s *Store) CreateSession(ctx context.Context, title string) (string, error) {
	return s.underlying.CreateSession(ctx, title)
}

// LoadMessages 核心：先 Redis，后 MySQL（Cache-Aside 读路径）。
func (s *Store) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	// ---------- 第 1 步：尝试读 Redis ----------
	if s.rdb != nil {
		data, err := s.rdb.Get(ctx, s.msgKey(sessionID)).Bytes()
		if err == nil {
			// 命中缓存：JSON 反序列化成 []llm.Message
			var msgs []llm.Message
			if err := json.Unmarshal(data, &msgs); err == nil {
				return msgs, nil
			}
		} else if err != goredis.Nil {
			// goredis.Nil = key 不存在，属于正常未命中
			// 其他错误（Redis 挂了）→ 打印日志，降级走 MySQL
			fmt.Printf("[cached] redis get 失败，降级 MySQL: %v\n", err)
		}
	}

	// ---------- 第 2 步：缓存未命中，读 MySQL ----------
	msgs, err := s.underlying.LoadMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// ---------- 第 3 步：回填 Redis（下次读就快了）----------
	s.setCache(ctx, sessionID, msgs)
	return msgs, nil
}

// AppendMessage 写路径：先写 MySQL，再删 Redis 缓存。
//
// 为什么删除而不是更新缓存？
//   实现简单、不会脏数据。下次 LoadMessages 会从 MySQL 重建缓存。
func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg llm.Message) error {
	if err := s.underlying.AppendMessage(ctx, sessionID, msg); err != nil {
		return err
	}
	s.invalidate(ctx, sessionID)
	return nil
}

// DeleteSession 删 MySQL 记录 + 清 Redis 缓存。
func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	if err := s.underlying.DeleteSession(ctx, sessionID); err != nil {
		return err
	}
	s.invalidate(ctx, sessionID)
	return nil
}

// ListSessions 列表查询走 MySQL（会话列表变化不频繁，不值得缓存）。
func (s *Store) ListSessions(ctx context.Context, limit int) ([]store.Session, error) {
	return s.underlying.ListSessions(ctx, limit)
}

// setCache 把消息列表 JSON 序列化后写入 Redis，并设置 TTL。
func (s *Store) setCache(ctx context.Context, sessionID string, msgs []llm.Message) {
	if s.rdb == nil {
		return
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return
	}
	// SET key value EX ttl —— 过期后自动删除，避免占满内存
	_ = s.rdb.Set(ctx, s.msgKey(sessionID), data, s.ttl).Err()
}

// invalidate 删除缓存（写操作后调用）。
func (s *Store) invalidate(ctx context.Context, sessionID string) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, s.msgKey(sessionID)).Err()
}
