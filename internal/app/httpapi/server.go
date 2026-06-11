// Package httpapi 提供 Agent 的 HTTP API（第 8 步）。
package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/app"
	"github.com/zhjing1019/goAiAgent/internal/observability"
)

// Server HTTP 服务。
type Server struct {
	// 应用实例
	app    *app.App
	// 路由
	mux    *http.ServeMux
	// 服务器
	server *http.Server
}

// New 创建 HTTP 服务（路由已注册，尚未监听端口）。
func New(application *app.App) *Server {
	s := &Server{app: application, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Addr 返回监听地址。
func (s *Server) Addr() string {
	if s.server != nil {
		return s.server.Addr
	}
	return ""
}

// ListenAndServe 启动 HTTP 服务并阻塞。
func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.withMiddleware(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("🚀 HTTP API 已启动: http://%s\n", addr)
	s.printRoutes()
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// withMiddleware 添加中间件（可观测性 + CORS + Redis 限流）。
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	h := next
	h = rateLimitMiddleware(s.app.RateLimiter())(h)
	h = observability.Middleware(h)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// printRoutes 打印路由
func (s *Server) printRoutes() {
	fmt.Println("   GET  /metrics")
	fmt.Println("   GET  /api/health")
	fmt.Println("   POST /api/chat")
	fmt.Println("   GET  /api/sessions")
	fmt.Println("   POST /api/knowledge/search")
}
