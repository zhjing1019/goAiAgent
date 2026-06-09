package milvus

import (
	"context"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/rag"
)

// OpenFromEnv 从环境变量打开 Milvus 知识库。
//
// 若未配置 MILVUS_ADDR 或 EMBEDDING_API_KEY，返回 (nil, nil) 表示 RAG 未启用。
func OpenFromEnv(ctx context.Context) (rag.KnowledgeBase, error) {
	milvusCfg, err := config.LoadMilvus()
	if err != nil {
		return nil, err
	}
	if !milvusCfg.Enabled() {
		return nil, nil
	}

	embedCfg, err := config.LoadEmbedding()
	if err != nil {
		return nil, err
	}
	if !embedCfg.Enabled() {
		return nil, fmt.Errorf("已配置 MILVUS_ADDR，但未配置 EMBEDDING_API_KEY（RAG 需要 Embedding 服务）")
	}

	embedder, err := rag.NewEmbedderFromConfig(embedCfg)
	if err != nil {
		return nil, err
	}

	return Open(ctx, milvusCfg, embedder)
}
