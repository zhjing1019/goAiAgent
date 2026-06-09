package rag

import (
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	lcopenai "github.com/tmc/langchaingo/llms/openai"
	"github.com/zhjing1019/goAiAgent/internal/config"
)

// NewEmbedderFromConfig 创建向量化客户端（OpenAI 兼容 /embeddings 接口）。
//
// 可用：OpenAI、SiliconFlow、阿里云等任何兼容 OpenAI Embedding 的服务。
func NewEmbedderFromConfig(cfg config.EmbeddingConfig) (embeddings.Embedder, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("Embedding 配置不完整")
	}

	llm, err := lcopenai.New(
		lcopenai.WithToken(cfg.APIKey),
		lcopenai.WithBaseURL(cfg.BaseURL),
		lcopenai.WithEmbeddingModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("create embedding client: %w", err)
	}
	return embeddings.NewEmbedder(llm)
}
