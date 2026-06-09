package milvus

import (
	"context"
	"os"
	"testing"

	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/rag"
)

func openTestKB(t *testing.T) *KB {
	t.Helper()
	if os.Getenv("MILVUS_ADDR") == "" || os.Getenv("EMBEDDING_API_KEY") == "" {
		t.Skip("MILVUS_ADDR or EMBEDDING_API_KEY not set")
	}
	milvusCfg, err := config.LoadMilvus()
	if err != nil {
		t.Fatal(err)
	}
	embedCfg, err := config.LoadEmbedding()
	if err != nil {
		t.Fatal(err)
	}
	embedder, err := rag.NewEmbedderFromConfig(embedCfg)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := Open(context.Background(), milvusCfg, embedder)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return kb
}

func TestKBAddSearch(t *testing.T) {
	kb := openTestKB(t)
	ctx := context.Background()

	unique := "go-agent-rag-test-" + t.Name()
	if err := kb.Add(ctx, unique+" 项目使用 Milvus 存储向量。", "integration"); err != nil {
		t.Fatal(err)
	}

	chunks, err := kb.Search(ctx, unique+" Milvus", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected search results")
	}
}
