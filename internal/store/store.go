// Package store 定义对话持久化接口（第 5 步）。
//
// Agent 只依赖 SessionStore 接口，不关心底层是 MySQL 还是其他数据库。
package store

import (
	"context"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

// Session 会话摘要（列表展示用）。
type Session struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SessionStore 对话存储接口。
type SessionStore interface {
	// Migrate 创建/更新表结构。
	Migrate(ctx context.Context) error

	// CreateSession 新建会话，返回 session_id。
	CreateSession(ctx context.Context, title string) (string, error)

	// LoadMessages 加载某会话的全部消息（按顺序）。
	LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error)

	// AppendMessage 追加一条消息。
	AppendMessage(ctx context.Context, sessionID string, msg llm.Message) error

	// DeleteSession 删除会话及其消息。
	DeleteSession(ctx context.Context, sessionID string) error

	// ListSessions 列出最近会话。
	ListSessions(ctx context.Context, limit int) ([]Session, error)
}
