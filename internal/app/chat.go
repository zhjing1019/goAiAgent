package app

import (
	"context"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/agent"
)

// ChatResult 一次 HTTP/API 对话结果。
type ChatResult struct {
	Reply     string
	SessionID string
}

// RunChat 处理带 session_id 的对话（HTTP API 用）。
//
// 规则：
//   - 有 MySQL + 提供 session_id → 从数据库加载历史后继续
//   - 有 MySQL + 无 session_id   → 创建新会话
//   - 无 MySQL                  → 复用 CLI 共享 Agent（单用户内存模式）
func (a *App) RunChat(ctx context.Context, sessionID, message string) (ChatResult, error) {
	if message == "" {
		return ChatResult{}, fmt.Errorf("message 不能为空")
	}

	if a.store != nil {
		ag, err := a.spawnAgent(sessionID)
		if err != nil {
			return ChatResult{}, err
		}
		if sessionID != "" {
			if err := ag.LoadSession(ctx, sessionID); err != nil {
				return ChatResult{}, err
			}
		}
		reply, err := ag.Run(ctx, message)
		if err != nil {
			return ChatResult{}, err
		}
		return ChatResult{Reply: reply, SessionID: ag.SessionID()}, nil
	}

	if sessionID != "" {
		return ChatResult{}, fmt.Errorf("未启用 MySQL，不支持指定 session_id")
	}
	reply, err := a.agent.Run(ctx, message)
	if err != nil {
		return ChatResult{}, err
	}
	return ChatResult{Reply: reply, SessionID: a.agent.SessionID()}, nil
}

func (a *App) spawnAgent(sessionID string) (*agent.Agent, error) {
	cfg := a.agentTpl
	cfg.SessionID = sessionID
	return agent.New(cfg)
}
