package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/rag"
)

type mockKB struct {
	added   []string
	results []rag.Chunk
}

func (m *mockKB) Add(_ context.Context, content, _ string) error {
	m.added = append(m.added, content)
	return nil
}

func (m *mockKB) Search(_ context.Context, _ string, _ int) ([]rag.Chunk, error) {
	return m.results, nil
}

func TestSearchKnowledgeTool(t *testing.T) {
	kb := &mockKB{
		results: []rag.Chunk{
			{Content: "Go Agent 支持 RAG", Source: "doc", Score: 0.9},
		},
	}
	tool := SearchKnowledgeTool{KB: kb}
	out, err := tool.Execute(context.Background(), `{"query":"RAG","top_k":1}`)
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Results []struct {
			Content string  `json:"content"`
			Score   float32 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Content == "" {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestAddKnowledgeTool(t *testing.T) {
	kb := &mockKB{}
	tool := AddKnowledgeTool{KB: kb}
	out, err := tool.Execute(context.Background(), `{"content":"记住这句话","source":"test"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"ok":true}` {
		t.Fatalf("unexpected: %s", out)
	}
	if len(kb.added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(kb.added))
	}
}
