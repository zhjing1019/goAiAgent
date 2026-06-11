// Package llm 负责和大模型（LLM）通信。
//
// 第 2 步：types.go  定义我们项目自己的请求/响应结构体
// 第 3 步：client.go  真正发 HTTP 请求调用 DeepSeek
//          convert.go  把我们的结构体 ↔ langchaingo 结构体互相转换
//
// 为什么要分 types 和 client？
//   - types：业务层统一数据结构（Agent、MySQL 记忆都用它）
//   - client：只关心「怎么调 API」
package llm

import (
	"context"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/config"
	lcopenai "github.com/tmc/langchaingo/llms/openai"
)

// Client 是对 DeepSeek 的封装，对外只暴露 Chat 方法。
//
// 内部持有 langchaingo 的 LLM 实例（lcopenai.LLM），它帮我们处理 HTTP 请求细节。
type Client struct {
	model string         // 默认模型名，Chat 时如果请求里没指定 model 就用它
	llm   *lcopenai.LLM // langchaingo 的 OpenAI 兼容客户端（DeepSeek 也能用）
}

// ClientConfig 手动创建 Client 时传入的配置（不一定从环境变量读）。
type ClientConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewClient 根据配置创建 Client。
//
// 典型用法：
//
//	client, err := llm.NewClient(llm.ClientConfig{
//	    APIKey:  "sk-xxx",
//	    BaseURL: "https://api.deepseek.com/v1",
//	    Model:   "deepseek-chat",
//	})
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("APIKey 不能为空")
	}

	// 补默认值
	if cfg.BaseURL == "" {
		cfg.BaseURL = config.NormalizeOpenAIBaseURL("https://api.deepseek.com")
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}

	// lcopenai.New 创建 langchaingo 客户端
	// WithToken   → 设置 API Key（相当于 HTTP Header: Authorization: Bearer sk-xxx）
	// WithBaseURL → 设置 API 根地址
	// WithModel   → 设置默认模型
	llm, err := lcopenai.New(
		lcopenai.WithToken(cfg.APIKey),
		lcopenai.WithBaseURL(cfg.BaseURL),
		lcopenai.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("create deepseek client: %w", err)
	}

	return &Client{model: cfg.Model, llm: llm}, nil
}

// NewClientFromEnv 从环境变量创建 Client（最常用，推荐）。
//
// 等价于：config.LoadDeepSeek() → NewClient(...)
func NewClientFromEnv() (*Client, error) {
	cfg, err := config.LoadDeepSeek()
	if err != nil {
		return nil, err
	}
	return NewClient(ClientConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	})
}

// Chat 向 DeepSeek 发送一次聊天请求，拿到回复。
//
// 参数说明：
//   - ctx：上下文，用于超时/取消（例如 ctx, cancel := context.WithTimeout(...)）
//   - req：第 2 步定义的 ChatRequest（消息列表、工具、温度等）
//
// 返回：
//   - *ChatResponse：模型回复（含文本、tool_calls、token 用量）
//   - error：网络错误、鉴权失败、API 报错等
//
// 一次 Chat 的内部流程（4 步）：
//   1. 把我们的 []Message 转成 langchaingo 格式
//   2. 把 Tools、Temperature 等转成 langchaingo 的 CallOption
//   3. 调用 llm.GenerateContent() 发 HTTP 请求
//   4. 把 langchaingo 的响应转回我们的 ChatResponse
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 如果调用方没指定 model，用 Client 创建时的默认 model
	if req.Model == "" {
		req.Model = c.model
	}

	// 第 1 步：格式转换（我们的 Message → langchaingo MessageContent）
	msgs, err := toLangChainMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	// 第 2+3 步：带选项调用 langchaingo
	// buildCallOptions(req) 返回 []CallOption，例如 WithTemperature、WithTools
	// "..." 表示把切片展开成多个参数
	resp, err := c.llm.GenerateContent(ctx, msgs, buildCallOptions(req)...)
	if err != nil {
		return nil, fmt.Errorf("deepseek chat: %w", err)
	}

	// 第 4 步：格式转换（langchaingo ContentResponse → 我们的 ChatResponse）
	return fromLangChainResponse(req.Model, resp), nil
}
