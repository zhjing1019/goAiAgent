package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/rag"
)

// IngestReport 批量导入结果。
type IngestReport struct {
	Files  int
	Chunks int
}

// IngestDir 将目录下 .md / .txt 文档写入知识库。
func (a *App) IngestDir(ctx context.Context, dir string) (IngestReport, error) {
	if a.kb == nil {
		return IngestReport{}, fmt.Errorf("RAG 未启用")
	}
	files, err := listDocFiles(dir)
	if err != nil {
		return IngestReport{}, err
	}
	if len(files) == 0 {
		return IngestReport{}, fmt.Errorf("目录 %s 下没有 .md / .txt 文件", dir)
	}

	var report IngestReport
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return report, fmt.Errorf("读取 %s: %w", path, err)
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			continue
		}
		source := filepath.Base(path)
		chunks := rag.SplitText(text, 500)
		if err := a.addKnowledgeWithRetry(ctx, text, source); err != nil {
			return report, fmt.Errorf("写入 %s: %w", source, err)
		}
		report.Files++
		report.Chunks += len(chunks)
		time.Sleep(2 * time.Second) // Milvus Standalone 限流保护
	}
	return report, nil
}

// SearchKnowledge 检索知识库。
func (a *App) SearchKnowledge(ctx context.Context, query string, topK int) ([]rag.Chunk, error) {
	if a.kb == nil {
		return nil, fmt.Errorf("RAG 未启用")
	}
	return a.kb.Search(ctx, query, topK)
}

// AddKnowledge 写入一条知识。
func (a *App) AddKnowledge(ctx context.Context, content, source string) error {
	if a.kb == nil {
		return fmt.Errorf("RAG 未启用")
	}
	return a.kb.Add(ctx, content, source)
}

// SeedKnowledge 写入示例知识，便于测试 RAG。
func (a *App) SeedKnowledge(ctx context.Context) error {
	if a.kb == nil {
		return fmt.Errorf("RAG 未启用")
	}
	docs := []struct {
		content string
		source  string
	}{
		{"Go Agent 项目支持多轮对话、工具调用、MySQL 持久化和 Milvus RAG。", "项目简介"},
		{"DeepSeek 是 OpenAI 兼容的大模型 API，本项目通过 langchaingo 调用。", "模型说明"},
		{"Agent 循环：用户输入 → LLM → 有 tool_calls 就执行工具 → 再调 LLM → 直到返回文本。", "架构说明"},
	}
	for _, d := range docs {
		if err := a.kb.Add(ctx, d.content, d.source); err != nil {
			return err
		}
	}
	return nil
}

// addKnowledgeWithRetry 写入知识，重试5次，每次间隔3秒，如果失败则返回错误
func (a *App) addKnowledgeWithRetry(ctx context.Context, content, source string) error {
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		err = a.kb.Add(ctx, content, source)
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

// listDocFiles 列出目录下的 .md 和 .txt 文件
func listDocFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录: %w", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".md" || ext == ".txt" {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}
