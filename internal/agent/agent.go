package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"
)

// Agent 带工具调用能力的对话 Agent（支持内存 + MySQL 多轮记忆）。
//
// 一次 Run 的流程（LangGraph 风格循环）：
//
//	loop:
//	  1. 把 history + 新用户消息发给 LLM（附带 tools）
//	  2. 若模型返回 tool_calls → 执行工具 → 把结果追加到 history → 继续 loop
//	  3. 若模型返回纯文本 → 结束，返回答案
type Agent struct {
	client       *llm.Client
	tools        *Registry
	store        store.SessionStore // 可选：MySQL 持久化
	sessionID    string             // 当前会话 ID
	systemPrompt string
	maxSteps     int
	messages     []llm.Message // 多轮记忆（不含 system）
}

// Config Agent 配置。
type Config struct {
	Client       *llm.Client
	Tools        *Registry
	Store        store.SessionStore // 可选：传入 mysql.Store
	SessionID    string             // 可选：继续已有会话
	SystemPrompt string
	MaxSteps     int // 防止无限循环，默认 10
}

// New 创建 Agent。
func New(cfg Config) (*Agent, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("Client 不能为空")
	}
	if cfg.Tools == nil {
		return nil, fmt.Errorf("Tools 不能为空")
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = "你是一个 helpful 的 AI 助手。需要时请调用工具，拿到结果后再回答用户。"
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}
	return &Agent{
		client:       cfg.Client,
		tools:        cfg.Tools,
		store:        cfg.Store,
		sessionID:    cfg.SessionID,
		systemPrompt: cfg.SystemPrompt,
		maxSteps:     cfg.MaxSteps,
		messages:     []llm.Message{},
	}, nil
}

// Run 处理用户输入，返回最终文本答案。
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	debugLog("用户输入: %s", userInput)

	if err := a.ensureSession(ctx, userInput); err != nil {
		return "", err
	}

	userMsg := llm.NewUserMessage(userInput)
	if err := a.appendMessage(ctx, userMsg); err != nil {
		return "", err
	}

	for step := 1; step <= a.maxSteps; step++ {
		req := llm.ChatRequest{
			Messages:   a.buildMessagesForLLM(),
			Tools:      a.tools.LLMTools(),
			ToolChoice: "auto",
		}

		if step == 1 {
			for _, tool := range req.Tools {
				debugLog("可用工具: %s — %s", tool.Function.Name, tool.Function.Description)
			}
		}
		debugLog("step %d/%d: 调用 LLM（history=%d）", step, a.maxSteps, len(a.messages))

		resp, err := a.client.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("step %d chat: %w", step, err)
		}

		if resp.Usage != nil {
			debugLog("step %d: tokens prompt=%d completion=%d total=%d",
				step, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}

		msg := resp.AssistantMessage()
		if msg == nil {
			return "", fmt.Errorf("step %d: 模型无回复", step)
		}

		debugLog("step %d: LLM 回复 content=%q tool_calls=%d", step, msg.Content, len(msg.ToolCalls))

		// 把 assistant 消息（可能含 tool_calls）记入 history
		if err := a.appendMessage(ctx, *msg); err != nil {
			return "", err
		}

		// 没有 tool_calls → 最终答案
		if !msg.HasToolCalls() {
			debugLog("step %d: 完成", step)
			return msg.Content, nil
		}

		// 有 tool_calls → 逐个执行，结果以 tool 消息回传
		for _, tc := range msg.ToolCalls {
			debugLog("step %d: 调用工具 %s args=%s", step, tc.Function.Name, tc.Function.Arguments)
			result, err := a.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				debugLog("step %d: 工具 %s 失败: %v", step, tc.Function.Name, err)
				result = toolErrorResult(err)
			} else {
				debugLog("step %d: 工具 %s 结果=%s", step, tc.Function.Name, result)
			}
			toolMsg := llm.NewToolResultMessage(tc.ID, tc.Function.Name, result)
			if err := a.appendMessage(ctx, toolMsg); err != nil {
				return "", err
			}
		}
	}

	return "", fmt.Errorf("超过最大循环步数 %d（可能工具调用陷入死循环）", a.maxSteps)
}

// buildMessagesForLLM 组装发给 LLM 的完整消息：system + 历史。
func (a *Agent) buildMessagesForLLM() []llm.Message {
	out := make([]llm.Message, 0, 1+len(a.messages))
	out = append(out, llm.NewSystemMessage(a.systemPrompt))
	out = append(out, a.messages...)
	return out
}

// Messages 返回当前对话历史（后面 MySQL 持久化会用）。
func (a *Agent) Messages() []llm.Message {
	cp := make([]llm.Message, len(a.messages))
	copy(cp, a.messages)
	return cp
}

// Reset 清空对话记忆；若启用了 MySQL，下次 Run 会创建新会话。
func (a *Agent) Reset() {
	a.messages = nil
	a.sessionID = ""
}

// ---------- 内置工具（演示用）----------

// GetCurrentTimeTool 返回当前时间。
type GetCurrentTimeTool struct{}
// 返回工具名称
func (GetCurrentTimeTool) Name() string        { return "get_current_time" }
// 返回工具描述
func (GetCurrentTimeTool) Description() string { return "获取当前日期和时间" }
// 返回工具参数	
func (GetCurrentTimeTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
// 执行工具
func (GetCurrentTimeTool) Execute(_ context.Context, _ string) (string, error) {
	return fmt.Sprintf(`{"time": %q}`, time.Now().Format("2006-01-02 15:04:05")), nil
}

// AddNumbersTool 两数相加。
type AddNumbersTool struct{}
// 返回工具名称
func (AddNumbersTool) Name() string        { return "add_numbers" }
// 返回工具描述
func (AddNumbersTool) Description() string { return "计算两个数字的和" }
// 返回工具参数
func (AddNumbersTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "number", "description": "第一个数"},
			"b": map[string]any{"type": "number", "description": "第二个数"},
		},
		"required": []string{"a", "b"},
	}
}
// 执行工具
func (AddNumbersTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	return fmt.Sprintf(`{"result": %g}`, args.A+args.B), nil
}

// MultiplyNumbersTool 两数相乘。
type MultiplyNumbersTool struct{}

func (MultiplyNumbersTool) Name() string        { return "multiply_numbers" }
func (MultiplyNumbersTool) Description() string { return "计算两个数字的乘积" }
func (MultiplyNumbersTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "number", "description": "第一个数"},
			"b": map[string]any{"type": "number", "description": "第二个数"},
		},
		"required": []string{"a", "b"},
	}
}
func (MultiplyNumbersTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	return fmt.Sprintf(`{"result": %g}`, args.A*args.B), nil
}

// DefaultRegistry 返回带演示工具的注册表。
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(GetCurrentTimeTool{})
	r.Register(AddNumbersTool{})
	r.Register(MultiplyNumbersTool{})
	return r
}
