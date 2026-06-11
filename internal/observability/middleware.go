// 第 2 课：HTTP 中间件（Middleware）。
//
// 中间件 = 插在「请求」和「业务 handler」之间的通用逻辑。
// 每个请求都会先过中间件，再到达 handleChat / handleHealth 等。
//
// 洋葱模型（从外到内执行，从内到外返回）：
//
//	请求进入 →
//	  recover（防 panic）
//	    → request_id（生成/读取 ID）
//	      → instrument（记访问日志 + 耗时；指标在第 3 课详讲）
//	        → rateLimit / CORS（在 server.go 里再包一层）
//	          → handleChat（你的业务）
//	响应返回 ←
package observability

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware 把多个中间件组合成一个 Handler。
//
// 注意组装顺序：最后包的 recover 最先执行（在最外层保护整个调用链）。
func Middleware(next http.Handler) http.Handler {
	h := next
	h = instrumentMiddleware(h)  // 最内层（紧贴业务）：请求结束后打访问日志
	h = requestIDMiddleware(h)   // 中间层：生成 request_id
	h = recoverMiddleware(h)       // 最外层：捕获 panic
	return h
}

// recoverMiddleware 防止 handler 里 panic 导致整个进程崩溃。
//
// 没有它时：某个 nil 指针 → 整个 agent-server 进程退出 → 所有用户受影响
// 有了它：返回 500 + 打 error 日志，进程继续运行
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("http panic recovered",
					"request_id", RequestIDFromContext(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// requestIDMiddleware 为每个请求分配 request_id。
//
// 逻辑：
//  1. 客户端传了 X-Request-ID 头 → 沿用（方便和前端/网关串联）
//  2. 没传 → 服务端生成新的 UUID
//  3. 写入 r.Context()，业务代码可通过 RequestIDFromContext 读取
//  4. 写入响应头 X-Request-ID，客户端也能拿到（Postman 的 Headers 里可见）
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = NewRequestID()
		}
		ctx := WithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusRecorder 包装 http.ResponseWriter，拦截真实 HTTP 状态码。
//
// 为什么需要它？
//   标准 ResponseWriter 不告诉你最后返回的是 200 还是 400，
//   中间件想在请求结束后记「status=400」就必须自己记一份。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// instrumentMiddleware 访问日志（Access Log）：每个请求结束后打一行 slog。
//
// 这是企业最常见的 HTTP 日志，有时叫 access log：
//   谁、什么时间、什么接口、返回码多少、花了多久
//
// 同时更新 Prometheus 指标（第 3 课详讲 HTTPRequestsTotal 等）。
func instrumentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		elapsed := time.Since(start)
		status := rec.status
		statusStr := fmt.Sprintf("%d", status)

		// 第 1 课学的 slog：这里自动为每个 HTTP 请求打一行结构化日志
		slog.Info("http request",
			"request_id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"latency_ms", elapsed.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)

		// 第 3 课：更新 Prometheus 指标
		// Inc()     → 请求计数 +1
		// Observe() → 把本次耗时记入直方图
		HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(r.Method, r.URL.Path).Observe(elapsed.Seconds())
	})
}
