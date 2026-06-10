// 从 Milvus 知识库检索，用于验证入库结果。
//
// 用法：make kb-search QUERY='企业级 Milvus 怎么部署'
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhjing1019/goAiAgent/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run ./cmd/kb-search <问题>")
		os.Exit(1)
	}
	query := os.Args[1]

	ctx := context.Background()
	application, err := app.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	if !application.Status().RAGEnabled {
		log.Fatal("RAG 未启用，请配置 MILVUS_ADDR 和 EMBEDDING_API_KEY")
	}

	chunks, err := application.SearchKnowledge(ctx, query, 5)
	if err != nil {
		log.Fatalf("检索失败: %v", err)
	}
	if len(chunks) == 0 {
		fmt.Println("（无匹配结果）")
		return
	}
	fmt.Printf("🔍 查询: %s\n\n", query)
	for i, c := range chunks {
		fmt.Printf("--- [%d] score=%.3f source=%s ---\n%s\n\n", i+1, c.Score, c.Source, c.Content)
	}
}
