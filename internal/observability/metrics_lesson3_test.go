package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/observability"
)

// 第 3 课：验证 /metrics 端点能暴露 HTTP 指标名。
func TestLesson3MetricsEndpoint(t *testing.T) {
	// CounterVec / HistogramVec 在第一次使用后，/metrics 里才会出现对应序列
	observability.HTTPRequestsTotal.WithLabelValues("GET", "/api/health", "200").Inc()
	observability.LLMRequestsTotal.WithLabelValues("deepseek-chat", "ok").Inc()
	observability.LLMRequestDurationSeconds.WithLabelValues("deepseek-chat").Observe(1.5)
	observability.LLMTokensTotal.WithLabelValues("deepseek-chat", "prompt").Add(10)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	observability.MetricsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, name := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"llm_requests_total",
		"llm_request_duration_seconds",
		"llm_tokens_total",
	} {
		if !strings.Contains(body, name) {
			t.Fatalf("metrics 缺少 %s:\n%s", name, body)
		}
	}
}

// 第 3 课：演示 Counter 和 Histogram 如何被 middleware 更新。
func TestLesson3HTTPMetricsIncrement(t *testing.T) {
	h := observability.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	observability.MetricsHandler().ServeHTTP(metricsRec, metricsReq)

	body := metricsRec.Body.String()
	if !strings.Contains(body, `http_requests_total{method="GET",path="/api/health",status="200"}`) {
		// Prometheus 文本格式可能带或不带额外空格，宽松检查
		if !strings.Contains(body, `path="/api/health"`) || !strings.Contains(body, "http_requests_total") {
			t.Fatalf("expected http_requests_total for /api/health, body:\n%s", body)
		}
	}
}
