package httpapi

import (
	"net"
	"net/http"
	"strings"

	"github.com/zhjing1019/goAiAgent/internal/redisx"
)

// rateLimitMiddleware HTTP 中间件：在请求到达 handler 之前做限流检查。
//
// 执行顺序（server.go 里组装）：
//
//	请求 → CORS 中间件 → 限流中间件 → 具体 handler（如 handleChat）
//
// 只对 POST /api/chat 限流，健康检查等接口不受影响。
func rateLimitMiddleware(limiter *redisx.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 未启用限流，或非 chat 接口 → 直接放行
			if limiter == nil || r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
				next.ServeHTTP(w, r)
				return
			}

			// 用客户端 IP 作为限流维度（同一 IP 共享配额）
			ip := clientIP(r)
			ok, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "限流服务异常")
				return
			}
			if !ok {
				// HTTP 429 = Too Many Requests
				writeError(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP 从请求里提取真实客户端 IP。
//
// 优先级：X-Forwarded-For（nginx 反代）> X-Real-IP > RemoteAddr
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
