// Package milvus 用 Milvus 实现向量知识库（第 6 步）。
package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/entity"
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
	// BGE 等 Embedding 模型使用余弦相似度；langchaingo 默认 metric 为空会导致建索引失败。
	store, err := milvusstore.New(ctx, cfg.Addr,
		milvusstore.WithCollectionName(cfg.Collection),
		milvusstore.WithEmbedder(embedder),
		milvusstore.WithMetricType(entity.COSINE),
	)
	if err != nil {
		return nil, fmt.Errorf("init milvus store: %w", err)
	}
	return &KB{store: store}, nil
}

// Add 写入知识（长文本会自动切片）。
func (k *KB) Add(ctx context.Context, content, source string) error {
	// 将文本切成多个片段
	parts := rag.SplitText(content, 500)
	// 如果片段数量为0，则返回错误
	if len(parts) == 0 {
		return fmt.Errorf("content 不能为空")
	}
	// 如果来源为空，则设置为 "manual"
	if source == "" {
		source = "manual"
	}

	// 创建一个空数组，用于存储文档
	docs := make([]schema.Document, 0, len(parts))
	// 遍历片段，将片段转换为文档
	for i, part := range parts {
		// 创建一个元数据对象
		meta := map[string]any{
			"source": source,
		}
		// 如果片段数量大于1，则设置片段编号和片段数量
		if len(parts) > 1 {
			meta["chunk"] = i + 1
			meta["chunks"] = len(parts)
		}
		// 将片段转换为文档，并添加到文档数组中
		docs = append(docs, schema.Document{
			PageContent: part,
			Metadata:    meta,
		})
	}
	// 将文档添加到知识库中
	// 如果添加失败，则返回错误
	if _, err := k.store.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("add documents: %w", err)
	}
	return nil
}

// Search 语义检索。	
func (k *KB) Search(ctx context.Context, query string, topK int) ([]rag.Chunk, error) {
	// 如果 topK 小于等于0，则设置为3
	if topK <= 0 {
		topK = 3
	}
	// 使用相似度搜索，返回相似的文档
	docs, err := k.store.SimilaritySearch(ctx, query, topK)
	// 如果搜索失败，则返回错误
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}

	// 创建一个空数组，用于存储结果
	out := make([]rag.Chunk, 0, len(docs))
	// 遍历文档，将文档转换为结果
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
