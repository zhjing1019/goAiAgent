// Package config 负责读取「配置」。
//
// 为什么不把 API Key 写死在代码里？
//   - 安全：Key 不能提交到 GitHub
//   - 灵活：开发/测试/生产可以用不同 Key
//
// 第 3 步只放 DeepSeek 相关配置，后面 MySQL、Milvus 也会放这里。
package config

import (
	"fmt"
	"os"
	"strings"
)

// DeepSeekConfig 连接 DeepSeek 需要的 3 个核心参数。
//
// DeepSeek 的 API 格式和 OpenAI 兼容，所以 langchaingo 的 openai 客户端可以直接用。
type DeepSeekConfig struct {
	APIKey  string // 密钥，从 DEEPSEEK_API_KEY 读取
	BaseURL string // API 地址，例如 https://api.deepseek.com/v1
	Model   string // 模型名，例如 deepseek-chat
}

// LoadDeepSeek 从操作系统「环境变量」加载配置。
//
// 环境变量是什么？
//   在终端 export DEEPSEEK_API_KEY=xxx 后，Go 程序可以通过 os.Getenv 读到。
//
// 使用前先确保已设置：
//   export DEEPSEEK_API_KEY=sk-你的key
//   export DEEPSEEK_BASE_URL=https://api.deepseek.com
func LoadDeepSeek() (DeepSeekConfig, error) {
	// TrimSpace 去掉首尾空格，避免复制 Key 时多了空格导致鉴权失败
	apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if apiKey == "" {
		return DeepSeekConfig{}, fmt.Errorf("环境变量 DEEPSEEK_API_KEY 未设置")
	}

	baseURL := strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL"))
	if baseURL == "" {
		// 没设置就用 DeepSeek 官方默认地址
		baseURL = "https://api.deepseek.com"
	}

	model := strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL"))
	if model == "" {
		// 默认对话模型
		model = "deepseek-chat"
	}

	return DeepSeekConfig{
		APIKey:  apiKey,
		BaseURL: NormalizeOpenAIBaseURL(baseURL),
		Model:   model,
	}, nil
}

// NormalizeOpenAIBaseURL 把 Base URL 修正成 langchaingo 需要的格式。
//
// 原因：langchaingo 内部会这样拼最终请求地址：
//   {BaseURL} + "/chat/completions"
//
// 所以 BaseURL 必须是：https://api.deepseek.com/v1
// 而不是：https://api.deepseek.com
//
// 示例：
//   输入  https://api.deepseek.com
//   输出  https://api.deepseek.com/v1
func NormalizeOpenAIBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/") // 去掉末尾 /
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return baseURL
}
