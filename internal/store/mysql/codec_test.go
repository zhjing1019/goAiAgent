package mysql

import (
	"database/sql"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

func TestEncodeDecodeUserMessage(t *testing.T) {
	msg := llm.NewUserMessage("你好")
	content, toolCalls, toolCallID, toolName, err := encodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeMessage("user", content, toolCalls, sql.NullString{}, sql.NullString{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != "user" || got.Content != "你好" {
		t.Fatalf("unexpected: %+v", got)
	}
	if toolCallID != "" || toolName != "" {
		t.Fatalf("expected empty tool fields")
	}
}

func TestEncodeDecodeAssistantWithToolCalls(t *testing.T) {
	msg := llm.Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []llm.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "add_numbers",
					Arguments: `{"a":1,"b":2}`,
				},
			},
		},
	}
	content, toolCallsJSON, _, _, err := encodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeMessage("assistant", content, toolCallsJSON, sql.NullString{}, sql.NullString{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Function.Name != "add_numbers" {
		t.Fatalf("unexpected tool_calls: %+v", got.ToolCalls)
	}
}

func TestEncodeDecodeToolResult(t *testing.T) {
	msg := llm.NewToolResultMessage("call_1", "add_numbers", `{"result":3}`)
	content, toolCalls, toolCallID, toolName, err := encodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeMessage("tool", content, toolCalls,
		sql.NullString{String: toolCallID, Valid: true},
		sql.NullString{String: toolName, Valid: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ToolCallID != "call_1" || got.Name != "add_numbers" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestTruncateTitle(t *testing.T) {
	if truncateTitle("", 10) != "新对话" {
		t.Fatal("empty title")
	}
	long := truncateTitle("这是一段很长很长很长很长很长很长的标题", 5)
	if len([]rune(long)) != 8 { // 5 + "..."
		t.Fatalf("unexpected: %q", long)
	}
}
