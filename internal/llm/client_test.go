package llm

// client_test.go：Client 的自动化测试
//
// Go 测试约定：
//   - 文件名以 _test.go 结尾
//   - 函数名以 Test 开头
//   - 运行：go test ./internal/llm/... -v

import (
	"context"
	"os"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/config"
)

// TestNewClientFromEnvMissingKey：没设置 API Key 时必须报错。
func TestNewClientFromEnvMissingKey(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "") // 测试里临时清空环境变量
	_, err := NewClientFromEnv()
	if err == nil {
		t.Fatal("expected error when API key missing")
	}
}

// TestNormalizeBaseURL：验证 URL 会自动补上 /v1。
func TestNormalizeBaseURL(t *testing.T) {
	got := config.NormalizeOpenAIBaseURL("https://api.deepseek.com")
	want := "https://api.deepseek.com/v1"
	if got != want {
		t.Fatalf("unexpected base url: got=%s want=%s", got, want)
	}
}

// TestDeepSeekChatIntegration：真正调用 DeepSeek API 的集成测试。
//
// 默认跳过（不耗 API 额度），只有设置了 DEEPSEEK_API_KEY 才跑：
//   DEEPSEEK_API_KEY=sk-xxx go test ./internal/llm -run TestDeepSeekChatIntegration -v
func TestDeepSeekChatIntegration(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		t.Skip("set DEEPSEEK_API_KEY to run integration test")
	}

	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			NewSystemMessage("你是一个简洁助手"),
			NewUserMessage("用一句话介绍 Go 语言"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg := resp.AssistantMessage()
	if msg == nil || msg.Content == "" {
		t.Fatalf("empty response: %+v", resp)
	}
}
