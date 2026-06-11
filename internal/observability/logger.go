// Package observability 可观测性基础设施。
//
// ┌─────────────────────────────────────────────────────────────┐
// │  第 1 课：结构化日志（本文件 logger.go）                      │
// │  第 2 课：HTTP 中间件 + request_id（middleware.go）          │
// │  第 3 课：Prometheus 指标（metrics.go）                      │
// └─────────────────────────────────────────────────────────────┘
//
// 第 1 课要解决的问题：
//   以前用 fmt.Println 打日志 → 格式乱、没法按字段搜索、没有日志级别
//   企业里用 slog 打 JSON 日志 → 机器可读、可接入日志平台、可区分 info/error
package observability

import (
	"log/slog"
	"os"
	"strings"
)

// Init 初始化全局日志器（整个进程只调用一次，放在 main 函数最开头）。
//
// 调用后，项目里任何地方都可以：
//
//	slog.Info("服务启动", "port", 8080)
//	slog.Error("连接失败", "err", err)
//
// 日志会输出到 stdout（标准输出），一行一条 JSON，例如：
//
//	{"time":"...","level":"INFO","msg":"http request","method":"GET","path":"/api/health","status":200}
//
// 环境变量 LOG_LEVEL（可选，写在 .env.development）：
//   debug → 最详细，开发排查用
//   info  → 默认，记录正常业务流程
//   warn  → 警告（降级、重试等）
//   error → 只打错误
func Init() {
	// ① 读环境变量，决定「多详细的日志才输出」
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	// ② Handler = 日志的「输出格式 + 目的地」
	//    JSONHandler → 每行输出一条 JSON（企业标准做法）
	//    os.Stdout   → 打到终端；Docker/K8s 会收集 stdout 送到日志平台
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level, // 低于此级别的日志会被丢弃（如 Level=info 时不输出 debug）
	})

	// ③ 设为全局默认日志器，之后 slog.Info / slog.Error 都走这里
	slog.SetDefault(slog.New(handler))
}

// parseLogLevel 把字符串转成 slog 的级别枚举。
//
// 日志级别从低到高：debug < info < warn < error
// 设置 info 时，debug 不会输出，info/warn/error 会输出。
func parseLogLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // 没配置 LOG_LEVEL 时用 info
	}
}
