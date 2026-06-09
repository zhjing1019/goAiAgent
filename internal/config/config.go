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

// MilvusConfig Milvus 向量库配置（第 6 步 RAG）。
type MilvusConfig struct {
	Addr       string // MILVUS_ADDR，例如 127.0.0.1:19530
	Token      string // MILVUS_TOKEN，Zilliz Cloud 等需要，本地可留空
	Collection string // MILVUS_COLLECTION，默认 go_agent_kb
}

// LoadMilvus 从环境变量加载 Milvus 配置。
func LoadMilvus() (MilvusConfig, error) {
	LoadEnv()
	addr := strings.TrimSpace(os.Getenv("MILVUS_ADDR"))
	token := strings.TrimSpace(os.Getenv("MILVUS_TOKEN"))
	collection := strings.TrimSpace(os.Getenv("MILVUS_COLLECTION"))
	if collection == "" {
		collection = "go_agent_kb"
	}
	return MilvusConfig{
		Addr:       addr,
		Token:      token,
		Collection: collection,
	}, nil
}

// Enabled 是否配置了 Milvus。
func (c MilvusConfig) Enabled() bool {
	return c.Addr != ""
}

// EmbeddingConfig 向量化（Embedding）配置。
//
// 需要 OpenAI 兼容的 /embeddings 接口（OpenAI、SiliconFlow、阿里云等）。
type EmbeddingConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// LoadEmbedding 从环境变量加载 Embedding 配置。
//
// 环境变量：
//
//	EMBEDDING_API_KEY=sk-...
//	EMBEDDING_BASE_URL=https://api.openai.com/v1
//	EMBEDDING_MODEL=text-embedding-3-small
func LoadEmbedding() (EmbeddingConfig, error) {
	LoadEnv()
	apiKey := strings.TrimSpace(os.Getenv("EMBEDDING_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("EMBEDDING_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("EMBEDDING_MODEL"))
	if model == "" {
		model = "text-embedding-3-small"
	}
	return EmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: NormalizeOpenAIBaseURL(baseURL),
		Model:   model,
	}, nil
}

// Enabled 是否配置了 Embedding。
func (c EmbeddingConfig) Enabled() bool {
	return c.APIKey != ""
}
