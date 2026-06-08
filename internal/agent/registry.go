// Package agent 实现 Agent 核心：工具注册 + Tool Calling 循环。
//
// 对应 LangGraph 里的「图式执行」：
//
//	用户输入 → LLM → 要调工具吗？
//	                  ├─ 否 → 返回最终答案
//	                  └─ 是 → 执行工具 → 结果回传 LLM → 再循环
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

// Tool 是 Agent 可以调用的工具（函数）。
//
// 每个工具需要告诉模型 3 件事：名字、描述、参数格式（JSON Schema）。
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, argsJSON string) (string, error)
}

// Registry 工具注册表：名字 → 工具实现。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具（同名会覆盖）。
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// LLMTools 把所有工具转成 LLM API 需要的 []llm.Tool 格式。
func (r *Registry) LLMTools() []llm.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]llm.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, llm.NewFunctionTool(t.Name(), t.Description(), t.Parameters()))
	}
	return out
}

// Execute 按名字执行工具。
//
// argsJSON 是模型返回的参数 JSON 字符串，例如 {"city":"北京"}。
func (r *Registry) Execute(ctx context.Context, name, argsJSON string) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	return t.Execute(ctx, argsJSON)
}

// toolErrorResult 工具失败时，把错误包装成 JSON 字符串回传给模型。
func toolErrorResult(err error) string {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(b)
}
