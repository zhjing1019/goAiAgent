package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

// encodeMessage 把 llm.Message 转成数据库字段。
func encodeMessage(msg llm.Message) (content string, toolCallsJSON []byte, toolCallID, toolName string, err error) {
	content = msg.Content
	toolCallID = msg.ToolCallID
	toolName = msg.Name

	if len(msg.ToolCalls) > 0 {
		toolCallsJSON, err = json.Marshal(msg.ToolCalls)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("marshal tool_calls: %w", err)
		}
	}
	return content, toolCallsJSON, toolCallID, toolName, nil
}

// decodeMessage 从数据库字段还原 llm.Message。
func decodeMessage(role, content string, toolCallsJSON []byte, toolCallID, toolName sql.NullString) (llm.Message, error) {
	msg := llm.Message{
		Role:    role,
		Content: content,
	}
	if toolCallID.Valid {
		msg.ToolCallID = toolCallID.String
	}
	if toolName.Valid {
		msg.Name = toolName.String
	}
	if len(toolCallsJSON) > 0 {
		if err := json.Unmarshal(toolCallsJSON, &msg.ToolCalls); err != nil {
			return llm.Message{}, fmt.Errorf("unmarshal tool_calls: %w", err)
		}
	}
	return msg, nil
}

// truncateTitle 用首条用户消息做会话标题。
func truncateTitle(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "新对话"
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
