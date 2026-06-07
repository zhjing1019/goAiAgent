// Package llm 定义与大模型 API 交互的请求/响应结构（OpenAI 兼容格式）。
package llm

// ---------- 角色常量（多轮对话）----------

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ---------- 请求：Chat Completions ----------

// ChatRequest 发送给 LLM 的聊天请求。
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`

	// 工具调用：Agent 能力核心字段，omitempty 字段为空 / 零值时，JSON 不输出该字段，适配接口规范（尤其是大模型 API）
	Tools      []Tool `json:"tools,omitempty"`
	ToolChoice any    `json:"tool_choice,omitempty"` // "auto" | "none" | {"type":"function","function":{"name":"xxx"}}
	// 温度：0.0-1.0，越高越随机，越低越确定，默认 1.0
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
}

// Message 单条对话消息（支持纯文本 + 工具调用往返）。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`

	// assistant 发起工具调用时填充
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// tool 角色回传工具结果时填充
	ToolCallID string `json:"tool_call_id,omitempty"`
	Name       string `json:"name,omitempty"`
}

// Tool 工具定义（告诉模型有哪些 function 可调用）。
type Tool struct {
	Type     string      `json:"type"` // 固定 "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数 schema（JSON Schema 描述参数）。
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall 模型返回的工具调用指令。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // 固定 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 模型选中的函数及参数（Arguments 是 JSON 字符串）。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ---------- 响应：Chat Completions ----------

// ChatResponse LLM 返回的完整响应。
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object,omitempty"`
	Created int64    `json:"created,omitempty"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice 候选回复（通常只用 choices[0]）。
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // stop | tool_calls | length ...
}

// Usage token 用量统计。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ---------- 辅助构造（方便 Agent 组装请求）----------

// NewUserMessage 创建用户消息。
func NewUserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// NewSystemMessage 创建系统提示词。
func NewSystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// NewAssistantMessage 创建助手文本回复。
func NewAssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

// NewToolResultMessage 创建工具执行结果消息（回传给模型）。
func NewToolResultMessage(toolCallID, name, content string) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Name:       name,
		Content:    content,
	}
}

// NewFunctionTool 创建 function 类型工具定义。
func NewFunctionTool(name, description string, parameters map[string]any) Tool {
	return Tool{
		Type: "function",
		Function: FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// AssistantMessage 取第一条 assistant 回复（最常用）。
func (r *ChatResponse) AssistantMessage() *Message {
	if r == nil || len(r.Choices) == 0 {
		return nil
	}
	msg := r.Choices[0].Message
	return &msg
}

// HasToolCalls 判断模型是否要求调用工具。
func (m *Message) HasToolCalls() bool {
	return m != nil && len(m.ToolCalls) > 0
}
