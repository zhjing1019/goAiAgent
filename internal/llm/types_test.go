package llm

import (
	"encoding/json"
	"testing"
)

func TestChatRequestJSON(t *testing.T) {
	temp := 0.7
	req := ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			NewSystemMessage("你是一个 Go Agent 助手"),
			NewUserMessage("北京天气怎么样？"),
		},
		Tools: []Tool{
			NewFunctionTool("get_weather", "查询城市天气", map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string", "description": "城市名"},
				},
				"required": []string{"city"},
			}),
		},
		ToolChoice:  "auto",
		Temperature: &temp,
	}

	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty json")
	}
}

func TestToolResultMessage(t *testing.T) {
	msg := NewToolResultMessage("call_123", "get_weather", `{"temp":25}`)
	if msg.Role != RoleTool || msg.ToolCallID != "call_123" {
		t.Fatalf("unexpected tool message: %+v", msg)
	}
}
