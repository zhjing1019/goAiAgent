package observability_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/observability"
)

// 第 1 课：演示 slog 结构化日志的字段格式。
func TestLesson1StructuredLogging(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	slog.Info("用户发起对话", "session_id", "abc-123", "message_len", 12)
	slog.Error("MySQL 连接失败", "err", "connection refused")

	out := buf.String()
	if !strings.Contains(out, `"msg":"用户发起对话"`) {
		t.Fatalf("missing info log: %s", out)
	}
	if !strings.Contains(out, `"session_id":"abc-123"`) {
		t.Fatalf("missing structured field: %s", out)
	}
	if !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("missing error level: %s", out)
	}

	// Init 烟雾测试：确保可重复调用不 panic
	observability.Init()
}
