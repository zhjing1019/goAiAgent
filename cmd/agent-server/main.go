// 第 8 步：Agent HTTP API 服务入口
//
// 运行：
//
//	make run-server-dev
//	curl http://localhost:8080/api/health
//	curl -X POST http://localhost:8080/api/chat -H 'Content-Type: application/json' -d '{"message":"现在几点？"}'
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/app"
	"github.com/zhjing1019/goAiAgent/internal/app/httpapi"
	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/observability"
)

func main() {
	// 第 1 课：第一件事 —— 初始化结构化日志（后面所有 slog.Info/Error 才有效）
	observability.Init()
	slog.Info("进程启动", "service", "agent-server")

	ctx := context.Background()
	application, err := app.NewFromEnv(ctx)
	if err != nil {
		slog.Error("应用初始化失败", "err", err)
		os.Exit(1)
	}
	defer application.Close()

	application.Status().PrintStartup()

	addr, err := config.LoadHTTPAddr()
	if err != nil {
		slog.Error("读取 HTTP 配置失败", "err", err)
		os.Exit(1)
	}

	srv := httpapi.New(application)
	go func() {
		if err := srv.ListenAndServe(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号，正在优雅关闭")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("关闭 HTTP 服务失败", "err", err)
		os.Exit(1)
	}
	slog.Info("进程已退出")
}
