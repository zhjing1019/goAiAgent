// 第 3 课：Prometheus 指标（Metrics）。
//
// 和日志的区别（复习）：
//   日志 slog  → 记录「某一次发生了什么」（日记，查细节）
//   指标       → 记录「累计有多少、平均多快」（仪表盘，看趋势、做告警）
//
// 工作流程：
//   1. 业务代码里 .Inc() / .Observe() 更新数字
//   2. Prometheus 定期 GET /metrics 拉取
//   3. Grafana 画曲线图；超阈值时 Alertmanager 发钉钉/飞书
package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// promauto 会在进程启动时自动注册指标，/metrics 端点能直接暴露它们。
//
// 两种最常用的指标类型：
//
// Counter（计数器）
//   - 只增不减（像里程表）
//   - 适合：请求总数、错误总数、Token 消耗总量
//   - 操作：.Inc() 加 1，.Add(n) 加 n
//
// Histogram（直方图）
//   - 记录多次观测值，自动分桶统计分布
//   - 适合：接口耗时、LLM 延迟（可算 P50/P99）
//   - 操作：.Observe(秒数)

var (
	// ---------- HTTP 指标（在 middleware.go 的 instrumentMiddleware 里更新）----------

	// HTTPRequestsTotal 每个 HTTP 请求结束时 +1。
	//
	// 标签（labels）把数据分组，例如：
	//   http_requests_total{method="POST", path="/api/chat", status="200"} 42
	//   → POST /api/chat 成功返回了 42 次
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDurationSeconds 记录每个请求的耗时（秒）。
	//
	// Grafana 可据此画：「/api/chat P99 延迟是不是突然变慢了？」
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求耗时（秒）",
			Buckets: prometheus.DefBuckets, // 0.005s, 0.01s, ... 10s
		},
		[]string{"method", "path"},
	)

	// ---------- LLM 指标（在 internal/llm/client.go 的 Chat 里更新）----------

	// LLMRequestsTotal 每次调 DeepSeek +1。
	//
	// status 标签：ok | error
	// 告警示例：error 占比突然升高 → DeepSeek 可能挂了或 Key 失效
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "LLM API 调用总数",
		},
		[]string{"model", "status"},
	)

	// LLMRequestDurationSeconds 记录单次 LLM 调用耗时。
	//
	// LLM 通常比 HTTP 慢很多，所以桶设得更大：0.5s ~ 60s
	LLMRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "LLM API 调用耗时（秒）",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 40, 60},
		},
		[]string{"model"},
	)

	// LLMTokensTotal 累计 Token 消耗（直接关联 API 费用）。
	//
	// type 标签：prompt（输入）| completion（输出）
	// 告警示例：一天 prompt+completion 超过预算 → 发通知
	LLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_tokens_total",
			Help: "LLM Token 消耗总数",
		},
		[]string{"model", "type"},
	)
)

// MetricsHandler 返回 GET /metrics 的处理器。
//
// 访问 http://localhost:8080/metrics 可看到纯文本指标，例如：
//
//	http_requests_total{method="GET",path="/api/health",status="200"} 5
//	llm_tokens_total{model="deepseek-chat",type="prompt"} 1280
//
// 注意：/metrics 通常不经过限流中间件，但会经过访问日志（多一行 http request 日志）。
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
