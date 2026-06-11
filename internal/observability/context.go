// 第 2 课：request_id 与 context 传递。
//
// 问题：一次 HTTP 请求会经过 middleware → handler → Agent → MySQL，
//       日志散落在多处，怎么知道哪些日志属于同一次请求？
// 答案：给每次请求一个唯一 ID（request_id），放进 context，全程带着走。
package observability

import (
	"context"

	"github.com/google/uuid"
)

// ctxKey 自定义 context 的 key 类型。
//
// 为什么不用 string 做 key？
//   Go 官方建议用「未导出的自定义类型」避免和其他包冲突。
type ctxKey int

const requestIDKey ctxKey = 1

// NewRequestID 生成请求唯一 ID。
//
// 格式是 UUID，例如：a53f31c8-7a14-4204-b645-a3d52845b858
// 类似快递单号：查日志时搜这个 ID，能看到一次请求的全链路。
func NewRequestID() string {
	return uuid.NewString()
}

// WithRequestID 把 request_id 存入 context。
//
// context 像「手提袋」，函数调用链一路传下去：
//
//	middleware 放入 request_id
//	  → handler 从 r.Context() 取出
//	    → RunChat(ctx) 继续传
//	      → 更深处的 slog 也能取到同一个 ID
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext 从 context 读取 request_id。
//
// 用法（在 handler 或业务代码里）：
//
//	slog.Info("处理对话",
//	    "request_id", observability.RequestIDFromContext(r.Context()),
//	    "session_id", sessionID,
//	)
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}
