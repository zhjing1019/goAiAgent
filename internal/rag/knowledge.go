// Package rag 实现向量知识库 RAG（第 6 步）。
//
// Agent 只依赖 KnowledgeBase 接口，底层可以是 Milvus 或其他向量库。
package rag

import "context"

// Chunk 检索到的一条知识片段。
type Chunk struct {
	Content string  // 文本内容
	Source  string  // 来源（文件名、标签等）
	Score   float32 // 相似度分数（越高越相关）
}

// KnowledgeBase 向量知识库接口。
type KnowledgeBase interface {
	// Add 写入一条知识（内部会做向量化）。
	Add(ctx context.Context, content, source string) error

	// Search 按语义相似度检索 topK 条知识。
	Search(ctx context.Context, query string, topK int) ([]Chunk, error)
}
