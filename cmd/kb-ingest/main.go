// 批量将文档切分并写入 Milvus 知识库。
//
// 用法：make kb-ingest
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
		fmt.Println("用法: go run ./cmd/kb-ingest <文档目录>")
		fmt.Println("示例: make kb-ingest")
		os.Exit(1)
	}
	dir := os.Args[1]

	ctx := context.Background()
	application, err := app.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	application.Status().PrintStartup()
	if !application.Status().RAGEnabled {
		log.Fatal("RAG 未启用，请在 .env.development 中配置 MILVUS_ADDR 和 EMBEDDING_API_KEY")
	}

	report, err := application.IngestDir(ctx, dir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n🎉 完成：%d 个文件，共 %d 个向量切片已写入 Milvus\n", report.Files, report.Chunks)
	fmt.Println("验证检索: make kb-search QUERY='企业级 Milvus 怎么部署'")
}
