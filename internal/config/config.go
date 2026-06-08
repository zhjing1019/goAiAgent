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

	"github.com/joho/godotenv"
)

// DeepSeekConfig 连接 DeepSeek 需要的 3 个核心参数。
//
// DeepSeek 的 API 格式和 OpenAI 兼容，所以 langchaingo 的 openai 客户端可以直接用。
type DeepSeekConfig struct {
	APIKey  string // 密钥，从 DEEPSEEK_API_KEY 读取
	BaseURL string // API 地址，例如 https://api.deepseek.com/v1
	Model   string // 模型名，例如 deepseek-chat
}

// LoadEnv 尝试加载项目根目录的 .env 文件到环境变量。
//
// 重要：Go 默认不会自动读 .env！
//   只有调用 LoadEnv 后，os.Getenv 才能读到 .env 里的值。
//
// 如果 .env 不存在（例如生产环境），会静默跳过，不影响已 export 的系统环境变量。
func LoadEnv() {
	_ = godotenv.Load()
}

// LoadDeepSeek 加载 DeepSeek 配置。
//
// 读取顺序：
//  1. 先 LoadEnv() 加载 .env 文件
//  2. 再用 os.Getenv 读取（.env 或 export 设置的都可以）
func LoadDeepSeek() (DeepSeekConfig, error) {
	LoadEnv()

	// TrimSpace 去掉首尾空格，避免复制 Key 时多了空格导致鉴权失败
	apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if apiKey == "" {
		return DeepSeekConfig{}, fmt.Errorf("DEEPSEEK_API_KEY 未设置（请在 .env 或终端 export 中配置）")
	}

	baseURL := strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	model := strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL"))
	if model == "" {
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
func NormalizeOpenAIBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return baseURL
}

// MySQLConfig MySQL 连接配置。
type MySQLConfig struct {
	DSN string // 完整 DSN，优先使用
}

// LoadMySQL 从环境变量加载 MySQL 配置。
//
// 环境变量 MYSQL_DSN，示例：
//
//	MYSQL_DSN=root:password@tcp(127.0.0.1:3306)/go_agent?parseTime=true&charset=utf8mb4&loc=Local
//
// 若未设置 MYSQL_DSN，返回空配置（表示不使用 MySQL，Agent 仍可用内存记忆）。
func LoadMySQL() (MySQLConfig, error) {
	LoadEnv()
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	return MySQLConfig{DSN: dsn}, nil
}

// Enabled 是否配置了 MySQL。
func (c MySQLConfig) Enabled() bool {
	return c.DSN != ""
}
