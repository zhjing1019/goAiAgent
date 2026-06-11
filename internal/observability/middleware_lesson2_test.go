package observability_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/observability"
)

// 第 2 课：验证 request_id 贯穿 middleware → handler。
func TestLesson2RequestIDPropagation(t *testing.T) {
	var handlerSeenID string
	h := observability.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSeenID = observability.RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 场景 A：客户端不传 ID，服务端自动生成
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if handlerSeenID == "" {
		t.Fatal("handler 应能从 context 读到 request_id")
	}
	if rec.Header().Get("X-Request-ID") != handlerSeenID {
		t.Fatalf("响应头应返回同一个 request_id")
	}

	// 场景 B：客户端传入自定义 ID，响应头应原样返回（便于和网关/前端串联）
	customID := "my-trace-id-001"
	req2 := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	req2.Header.Set("X-Request-ID", customID)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Request-ID") != customID {
		t.Fatalf("应沿用客户端 X-Request-ID，got %s", rec2.Header().Get("X-Request-ID"))
	}
}
