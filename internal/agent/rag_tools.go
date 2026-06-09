package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/rag"
)

// rag_tools.go：RAG 知识库工具（第 6 步）。

// SearchKnowledgeTool 从向量知识库检索相关文档。
type SearchKnowledgeTool struct {
	KB rag.KnowledgeBase
}

func (SearchKnowledgeTool) Name() string { return "search_knowledge" }
func (SearchKnowledgeTool) Description() string {
	return "从向量知识库检索与问题相关的文档片段。当用户询问产品文档、公司政策、私有知识、已入库资料时使用。"
}
func (SearchKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "检索关键词或用户问题",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "返回条数，默认 3",
			},
		},
		"required": []string{"query"},
	}
}
func (t SearchKnowledgeTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	if t.KB == nil {
		return "", fmt.Errorf("知识库未启用")
	}
	var args struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("query 不能为空")
	}

	chunks, err := t.KB.Search(ctx, args.Query, args.TopK)
	if err != nil {
		return "", err
	}
	if len(chunks) == 0 {
		return `{"results":[]}`, nil
	}

	type item struct {
		Content string  `json:"content"`
		Source  string  `json:"source,omitempty"`
		Score   float32 `json:"score"`
	}
	items := make([]item, 0, len(chunks))
	for _, c := range chunks {
		items = append(items, item{Content: c.Content, Source: c.Source, Score: c.Score})
	}
	b, err := json.Marshal(map[string]any{"results": items})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// AddKnowledgeTool 向知识库添加文本。
type AddKnowledgeTool struct {
	KB rag.KnowledgeBase
}

func (AddKnowledgeTool) Name() string { return "add_knowledge" }
func (AddKnowledgeTool) Description() string {
	return "向向量知识库添加一段文本。当用户明确要求「记住」「保存到知识库」时使用。"
}
func (AddKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "要保存的文本内容",
			},
			"source": map[string]any{
				"type":        "string",
				"description": "来源标签，例如 产品手册、FAQ",
			},
		},
		"required": []string{"content"},
	}
}
func (t AddKnowledgeTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	if t.KB == nil {
		return "", fmt.Errorf("知识库未启用")
	}
	var args struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	if args.Content == "" {
		return "", fmt.Errorf("content 不能为空")
	}
	if err := t.KB.Add(ctx, args.Content, args.Source); err != nil {
		return "", err
	}
	return `{"ok":true}`, nil
}
