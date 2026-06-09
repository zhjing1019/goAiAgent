// 批量将文档切分并写入 Milvus 知识库。
//
// 用法：
//
//	APP_ENV=development go run ./cmd/kb-ingest testdata/knowledge
//	make kb-ingest
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/rag"
	ragmilvus "github.com/zhjing1019/goAiAgent/internal/rag/milvus"
)

func main() {
	fmt.Printf("🌍 %s\n", config.EnvSummary())

	if len(os.Args) < 2 {
		fmt.Println("用法: go run ./cmd/kb-ingest <文档目录>")
		fmt.Println("示例: make kb-ingest")
		os.Exit(1)
	}
	dir := os.Args[1]

	ctx := context.Background()
	kb, err := ragmilvus.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("RAG 初始化失败: %v", err)
	}
	if kb == nil {
		log.Fatal("RAG 未启用，请在 .env.development 中配置 MILVUS_ADDR 和 EMBEDDING_API_KEY")
	}

	files, err := listDocs(dir)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		log.Fatalf("目录 %s 下没有 .md / .txt 文件", dir)
	}

	var totalChunks int
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("读取 %s 失败: %v", path, err)
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			fmt.Printf("⏭️  跳过空文件: %s\n", path)
			continue
		}

		source := filepath.Base(path)
		chunks := rag.SplitText(text, 500)
		if err := addWithRetry(ctx, kb, text, source); err != nil {
			log.Fatalf("写入 %s 失败: %v", source, err)
		}
		totalChunks += len(chunks)
		fmt.Printf("✅ %s → %d 个切片\n", source, len(chunks))
		time.Sleep(2 * time.Second) // Milvus Standalone 有速率限制，间隔写入避免触发
	}

	fmt.Printf("\n🎉 完成：%d 个文件，共 %d 个向量切片已写入 Milvus\n", len(files), totalChunks)
	fmt.Println("验证检索: make kb-search QUERY='企业级 Milvus 怎么部署'")
}

func addWithRetry(ctx context.Context, kb interface {
	Add(context.Context, string, string) error
}, content, source string) error {
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		err = kb.Add(ctx, content, source)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "rate limit") {
			return err
		}
		wait := time.Duration(attempt) * 3 * time.Second
		fmt.Printf("⏳ %s 触发 Milvus 限流，%v 后重试 (%d/5)...\n", source, wait, attempt)
		time.Sleep(wait)
	}
	return err
}

func listDocs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录: %w", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".md" || ext == ".txt" {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out, nil
}
