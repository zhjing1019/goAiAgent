package agent

import (
	"context"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"
)

// persist.go：Agent 与 SessionStore（MySQL）的集成（第 5 步）。

// SessionID 返回当前会话 ID（未启用存储或未创建时为空）。
func (a *Agent) SessionID() string {
	return a.sessionID
}

// LoadSession 从数据库加载历史，恢复多轮记忆。
func (a *Agent) LoadSession(ctx context.Context, sessionID string) error {
	if a.store == nil {
		return fmt.Errorf("未启用 SessionStore")
	}
	msgs, err := a.store.LoadMessages(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	a.messages = msgs
	a.sessionID = sessionID
	debugLog("已加载会话 %s，共 %d 条消息", sessionID, len(msgs))
	return nil
}

// ListSessions 列出最近会话（需启用 MySQL）。
func (a *Agent) ListSessions(ctx context.Context, limit int) ([]store.Session, error) {
	if a.store == nil {
		return nil, fmt.Errorf("未启用 SessionStore")
	}
	return a.store.ListSessions(ctx, limit)
}

// ensureSession 首次对话时创建会话记录。
func (a *Agent) ensureSession(ctx context.Context, title string) error {
	if a.store == nil || a.sessionID != "" {
		return nil
	}
	id, err := a.store.CreateSession(ctx, title)
	if err != nil {
		return err
	}
	a.sessionID = id
	debugLog("新建会话: %s", id)
	return nil
}

// appendMessage 追加到内存，并可选写入 MySQL。
func (a *Agent) appendMessage(ctx context.Context, msg llm.Message) error {
	a.messages = append(a.messages, msg)
	if a.store == nil || a.sessionID == "" {
		return nil
	}
	if err := a.store.AppendMessage(ctx, a.sessionID, msg); err != nil {
		return fmt.Errorf("persist message: %w", err)
	}
	return nil
}
