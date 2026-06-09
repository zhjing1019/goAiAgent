// Package milvus 用 Milvus 实现向量知识库（第 6 步）。
package milvus

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	milvusstore "github.com/tmc/langchaingo/vectorstores/milvus/v2"
	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/rag"
)

// KB Milvus 知识库实现。
type KB struct {
	store milvusstore.Store
}

// Open 连接 Milvus 并初始化 collection。
func Open(ctx context.Context, cfg config.MilvusConfig, embedder embeddings.Embedder) (*KB, error) {
	store, err := milvusstore.New(ctx, cfg.Addr,
		milvusstore.WithCollectionName(cfg.Collection),
		milvusstore.WithEmbedder(embedder),
	)
	if err != nil {
		return nil, fmt.Errorf("init milvus store: %w", err)
	}
	return &KB{store: store}, nil
}

// Add 写入知识（长文本会自动切片）。
func (k *KB) Add(ctx context.Context, content, source string) error {
	parts := rag.SplitText(content, 500)
	if len(parts) == 0 {
		return fmt.Errorf("content 不能为空")
	}
	if source == "" {
		source = "manual"
	}

	docs := make([]schema.Document, 0, len(parts))
	for i, part := range parts {
		meta := map[string]any{
			"source": source,
		}
		if len(parts) > 1 {
			meta["chunk"] = i + 1
			meta["chunks"] = len(parts)
		}
		docs = append(docs, schema.Document{
			PageContent: part,
			Metadata:    meta,
		})
	}

	if _, err := k.store.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("add documents: %w", err)
	}
	return nil
}

// Search 语义检索。
func (k *KB) Search(ctx context.Context, query string, topK int) ([]rag.Chunk, error) {
	if topK <= 0 {
		topK = 3
	}
	docs, err := k.store.SimilaritySearch(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}

	out := make([]rag.Chunk, 0, len(docs))
	for _, doc := range docs {
		source, _ := doc.Metadata["source"].(string)
		out = append(out, rag.Chunk{
			Content: doc.PageContent,
			Source:  source,
			Score:   doc.Score,
		})
	}
	return out, nil
}
