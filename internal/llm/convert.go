package llm

// convert.go：格式转换层（Adapter / 适配器）
//
// 为什么需要这个文件？
//
//   我们项目有自己的结构体（types.go 里的 Message、ChatRequest）
//   langchaingo 也有自己的结构体（lc.MessageContent、lc.Tool）
//
//   两套结构体字段名、角色名不完全一样，所以需要「翻译」：
//     发请求前：我们的格式 → langchaingo 格式
//     收响应后：langchaingo 格式 → 我们的格式
//
//   好处：上层 Agent 代码只认 types.go，不用直接依赖 langchaingo。

import (
	"fmt"

	lc "github.com/tmc/langchaingo/llms" // 别名 lc，少写长包名
)

// ---------- 请求方向：我们的 Message → langchaingo ----------

// toLangChainMessages 批量转换消息列表。
func toLangChainMessages(msgs []Message) ([]lc.MessageContent, error) {
	out := make([]lc.MessageContent, 0, len(msgs))
	for _, msg := range msgs {
		mc, err := toLangChainMessage(msg)
		if err != nil {
			return nil, err
		}
		out = append(out, mc)
	}
	return out, nil
}

// toLangChainMessage 转换单条消息。
//
// 角色对照表（我们的 → langchaingo）：
//
//	RoleSystem    → ChatMessageTypeSystem
//	RoleUser      → ChatMessageTypeHuman
//	RoleAssistant → ChatMessageTypeAI
//	RoleTool      → ChatMessageTypeTool
func toLangChainMessage(msg Message) (lc.MessageContent, error) {
	switch msg.Role {

	case RoleSystem:
		// 系统提示词，例如「你是一个 Agent 助手」
		return lc.TextParts(lc.ChatMessageTypeSystem, msg.Content), nil

	case RoleUser:
		// 用户说的话
		return lc.TextParts(lc.ChatMessageTypeHuman, msg.Content), nil

	case RoleAssistant:
		// 模型的回复，可能有两种情况：
		//   A. 纯文本回复（Content 有值）
		//   B. 要求调用工具（ToolCalls 有值，Content 可能为空）
		parts := make([]lc.ContentPart, 0, 1+len(msg.ToolCalls))

		if msg.Content != "" {
			parts = append(parts, lc.TextPart(msg.Content))
		}
		for _, tc := range msg.ToolCalls {
			parts = append(parts, lc.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				FunctionCall: &lc.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments, // JSON 字符串
				},
			})
		}
		return lc.MessageContent{Role: lc.ChatMessageTypeAI, Parts: parts}, nil

	case RoleTool:
		// 工具执行结果，回传给模型
		// langchaingo 要求 tool 消息用 ToolCallResponse 包装
		return lc.MessageContent{
			Role: lc.ChatMessageTypeTool,
			Parts: []lc.ContentPart{
				lc.ToolCallResponse{
					ToolCallID: msg.ToolCallID, // 对应哪一次 tool_call
					Name:       msg.Name,       // 工具名
					Content:    msg.Content,    // 工具返回的 JSON 字符串
				},
			},
		}, nil

	default:
		return lc.MessageContent{}, fmt.Errorf("unsupported role: %s", msg.Role)
	}
}

// toLangChainTools 把我们的 Tool 列表转成 langchaingo 的 Tool 列表。
//
// 工具定义会随 ChatRequest 一起发给模型，告诉模型「你可以调用这些函数」。
func toLangChainTools(tools []Tool) []lc.Tool {
	out := make([]lc.Tool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, lc.Tool{
			Type: tool.Type, // 通常是 "function"
			Function: &lc.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters, // JSON Schema
			},
		})
	}
	return out
}

// buildCallOptions 把 ChatRequest 里的可选参数转成 langchaingo 的 CallOption。
//
// CallOption 是什么？
//   langchaingo 用「函数选项模式」设置 temperature、tools 等：
//   llms.WithTemperature(0.7) 返回一个 CallOption
//   多个 CallOption 传给 GenerateContent
func buildCallOptions(req ChatRequest) []lc.CallOption {
	opts := []lc.CallOption{}

	if req.Model != "" {
		opts = append(opts, lc.WithModel(req.Model))
	}
	if req.Temperature != nil {
		opts = append(opts, lc.WithTemperature(*req.Temperature))
	}
	if req.TopP != nil {
		opts = append(opts, lc.WithTopP(*req.TopP))
	}
	if req.MaxTokens != nil {
		opts = append(opts, lc.WithMaxTokens(*req.MaxTokens))
	}
	if len(req.Tools) > 0 {
		opts = append(opts, lc.WithTools(toLangChainTools(req.Tools)))
	}
	if req.ToolChoice != nil {
		// 匿名函数也是一种 CallOption：直接改内部 CallOptions 结构体
		opts = append(opts, func(o *lc.CallOptions) {
			o.ToolChoice = req.ToolChoice // 通常 "auto"
		})
	}
	return opts
}

// ---------- 响应方向：langchaingo → 我们的 ChatResponse ----------

// fromLangChainResponse 把 langchaingo 的响应转成我们的 ChatResponse。
func fromLangChainResponse(model string, resp *lc.ContentResponse) *ChatResponse {
	// 防御：空响应
	if resp == nil || len(resp.Choices) == 0 {
		return &ChatResponse{Model: model, Choices: []Choice{}}
	}
	// 容量：预分配足够空间，避免多次扩容
	choices := make([]Choice, 0, len(resp.Choices))
	// 遍历 langchaingo 的响应，转换每条选择
	for i, c := range resp.Choices {
		// langchaingo 把 assistant 回复放在 Content 和 ToolCalls 里
		msg := Message{Role: RoleAssistant, Content: c.Content}

		// 如果模型要求调工具，ToolCalls 不为空（第 4 步 Agent 会处理）
		for _, tc := range c.ToolCalls {
			fn := FunctionCall{}
			if tc.FunctionCall != nil {
				fn.Name = tc.FunctionCall.Name
				fn.Arguments = tc.FunctionCall.Arguments
			}
			// 添加工具调用
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:       tc.ID,
				Type:     tc.Type,
				Function: fn,
			})
		}

		// 添加选择
		choices = append(choices, Choice{
			Index:        i,
			Message:      msg,
			FinishReason: c.StopReason, // stop / tool_calls 等
		})
	}

	// 创建响应
	out := &ChatResponse{
		Model:   model,
		Choices: choices,
	}

	// Token 用量（langchaingo 放在 GenerationInfo map 里），转换每条选择	
	if len(resp.Choices) > 0 && resp.Choices[0].GenerationInfo != nil {
		info := resp.Choices[0].GenerationInfo
		out.Usage = &Usage{
			PromptTokens:     toInt(info["PromptTokens"]),
			CompletionTokens: toInt(info["CompletionTokens"]),
			TotalTokens:      toInt(info["TotalTokens"]),
		}
	}
	return out
}

// toInt 把 interface{} 安全转成 int。
//
// 为什么需要？
//   GenerationInfo 是 map[string]any，值可能是 int/int64/float64，统一转成 int。
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
