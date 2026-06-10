// Package cached 用 Redis 缓存 MySQL 会话消息（Cache-Aside 旁路缓存）。
//
// ┌─────────┐    ① 先查 Redis     ┌───────┐
// │ Agent   │ ─────────────────▶ │ Redis │ 命中 → 直接返回，不查 MySQL
// │LoadSession                   └───────┘
// └────┬────┘    ② 未命中
//      │ ──────────────────────▶ MySQL 查到后 ③ 写入 Redis
//      ▼
//
// 防护措施（本文件实现）：
//   - 缓存穿透：不存在的 session 缓存空值标记（短 TTL）
//   - 缓存雪崩：正常缓存 TTL 加随机抖动，避免同时过期
package cached

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"
)

// ErrSessionNotFound 会话不存在（含空值缓存命中）。
var ErrSessionNotFound = errors.New("session not found")

// 空值缓存标记：区分「会话不存在」与「会话存在但消息 JSON 为空数组」。
const nullCacheMarker = "NULL"

// 不存在会话的空值缓存时间（短 TTL，避免长期占坑）。
const nullCacheTTL = 5 * time.Minute

// sessionExistenceChecker 底层存储可选实现：判断 sessions 表是否有该 ID。
type sessionExistenceChecker interface {
	SessionExists(ctx context.Context, sessionID string) (bool, error)
}

// Store 装饰器模式：在 MySQL Store 外面包一层 Redis 缓存。
type Store struct {
	underlying store.SessionStore
	rdb        *goredis.Client
	ttl        time.Duration
	prefix     string
}

// New 用 Redis 包装已有的 SessionStore。
func New(underlying store.SessionStore, rdb *goredis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Store{
		underlying: underlying,
		rdb:        rdb,
		ttl:        ttl,
		prefix:     "goagent:",
	}
}

func (s *Store) msgKey(sessionID string) string {
	return s.prefix + "session:msgs:" + sessionID
}

func (s *Store) Migrate(ctx context.Context) error {
	return s.underlying.Migrate(ctx)
}

func (s *Store) CreateSession(ctx context.Context, title string) (string, error) {
	return s.underlying.CreateSession(ctx, title)
}

// LoadMessages 先 Redis，后 MySQL；含穿透防护与雪崩 TTL 抖动。
func (s *Store) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	if s.rdb != nil {
		data, err := s.rdb.Get(ctx, s.msgKey(sessionID)).Bytes()
		if err == nil {
			// 命中空值缓存 → 会话不存在，直接拦截，不再打 MySQL
			if string(data) == nullCacheMarker {
				return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
			}
			var msgs []llm.Message
			if err := json.Unmarshal(data, &msgs); err == nil {
				return msgs, nil
			}
		} else if err != goredis.Nil {
			fmt.Printf("[cached] redis get 失败，降级 MySQL: %v\n", err)
		}
	}

	msgs, err := s.underlying.LoadMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 穿透防护：会话 ID 在 sessions 表不存在 → 缓存 NULL 标记
	if len(msgs) == 0 {
		if checker, ok := s.underlying.(sessionExistenceChecker); ok {
			exists, err := checker.SessionExists(ctx, sessionID)
			if err != nil {
				return nil, err
			}
			if !exists {
				s.setNullCache(ctx, sessionID)
				return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
			}
		}
	}

	s.setCache(ctx, sessionID, msgs)
	return msgs, nil
}

func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg llm.Message) error {
	if err := s.underlying.AppendMessage(ctx, sessionID, msg); err != nil {
		return err
	}
	s.invalidate(ctx, sessionID)
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	if err := s.underlying.DeleteSession(ctx, sessionID); err != nil {
		return err
	}
	s.invalidate(ctx, sessionID)
	return nil
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]store.Session, error) {
	return s.underlying.ListSessions(ctx, limit)
}

// cacheTTL 在基础 TTL 上加随机抖动（约 ±10%），避免大量 key 同时过期引发雪崩。
func (s *Store) cacheTTL() time.Duration {
	if s.ttl <= 0 {
		return 24 * time.Hour
	}
	jitter := time.Duration(rand.Int63n(int64(s.ttl/5))) - s.ttl/10
	return s.ttl + jitter
}

func (s *Store) setCache(ctx context.Context, sessionID string, msgs []llm.Message) {
	if s.rdb == nil {
		return
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, s.msgKey(sessionID), data, s.cacheTTL()).Err()
}

func (s *Store) setNullCache(ctx context.Context, sessionID string) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Set(ctx, s.msgKey(sessionID), nullCacheMarker, nullCacheTTL).Err()
}

func (s *Store) invalidate(ctx context.Context, sessionID string) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, s.msgKey(sessionID)).Err()
}
