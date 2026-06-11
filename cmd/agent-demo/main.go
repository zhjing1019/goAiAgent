// 第 7 步演示：通过 internal/app 统一装配的 Agent CLI
//
// 运行：make run-dev  或  go run ./cmd/agent-demo
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zhjing1019/goAiAgent/internal/app"
)

func main() {
	// 创建一个上下文
	ctx := context.Background()
	// 创建一个应用实例
	application, err := app.NewFromEnv(ctx)
	// 如果创建应用实例失败，则退出
	if err != nil {
		log.Fatal(err)
	}
	// 延迟关闭应用实例
	defer application.Close()

	st := application.Status()
	st.PrintStartup()
	fmt.Println("🤖 Agent 已启动（输入 exit 退出）")
	st.PrintHelp()
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("再见！")
			break
		}
		if input == "reset" || input == "new" {
			application.Reset()
			fmt.Println("（对话已清空，下次将创建新会话）")
			continue
		}
		if input == "sessions" {
			handleSessions(ctx, application)
			continue
		}
		if strings.HasPrefix(input, "load ") {
			handleLoad(ctx, application, input)
			continue
		}
		if strings.HasPrefix(input, "kb ") {
			handleKB(ctx, application, input)
			continue
		}

		answer, err := application.Run(ctx, input)
		if err != nil {
			fmt.Println("错误:", err)
			continue
		}
		fmt.Println("Agent:", answer)
		if sid := application.SessionID(); sid != "" && st.MySQLEnabled {
			fmt.Printf("（会话 ID: %s）\n", sid)
		}
	}
}

func handleSessions(ctx context.Context, a *app.App) {
	if !a.Status().MySQLEnabled {
		fmt.Println("未启用 MySQL，无法列出会话")
		return
	}
	list, err := a.ListSessions(ctx, 10)
	if err != nil {
		fmt.Println("错误:", err)
		return
	}
	if len(list) == 0 {
		fmt.Println("（暂无会话）")
		return
	}
	for _, s := range list {
		fmt.Printf("  %s  %s  %s\n", s.ID[:8]+"...", s.Title, s.UpdatedAt.Format("2006-01-02 15:04"))
	}
}

func handleLoad(ctx context.Context, a *app.App, input string) {
	if !a.Status().MySQLEnabled {
		fmt.Println("未启用 MySQL，无法加载会话")
		return
	}
	id := strings.TrimSpace(strings.TrimPrefix(input, "load "))
	if id == "" {
		fmt.Println("用法: load <session_id>")
		return
	}
	if err := a.LoadSession(ctx, id); err != nil {
		fmt.Println("错误:", err)
		return
	}
	fmt.Printf("已加载会话 %s，共 %d 条历史消息\n", id[:8]+"...", a.MessageCount())
}

func handleKB(ctx context.Context, a *app.App, input string) {
	if !a.Status().RAGEnabled {
		fmt.Println("RAG 未启用，请配置 MILVUS_ADDR 和 EMBEDDING_API_KEY")
		return
	}
	rest := strings.TrimSpace(strings.TrimPrefix(input, "kb "))
	switch {
	case rest == "seed":
		if err := a.SeedKnowledge(ctx); err != nil {
			fmt.Println("错误:", err)
			return
		}
		fmt.Println("✅ 已写入 3 条示例知识，可问：「这个项目有什么功能？」")
	case strings.HasPrefix(rest, "ingest "):
		dir := strings.TrimSpace(strings.TrimPrefix(rest, "ingest "))
		if dir == "" {
			fmt.Println("用法: kb ingest <目录>")
			return
		}
		report, err := a.IngestDir(ctx, dir)
		if err != nil {
			fmt.Println("错误:", err)
			return
		}
		fmt.Printf("✅ 共导入 %d 个文件，%d 个向量切片\n", report.Files, report.Chunks)
	case strings.HasPrefix(rest, "add "):
		text := strings.TrimSpace(strings.TrimPrefix(rest, "add "))
		if text == "" {
			fmt.Println("用法: kb add <文本>")
			return
		}
		if err := a.AddKnowledge(ctx, text, "cli"); err != nil {
			fmt.Println("错误:", err)
			return
		}
		fmt.Println("✅ 已写入知识库")
	case strings.HasPrefix(rest, "search "):
		query := strings.TrimSpace(strings.TrimPrefix(rest, "search "))
		if query == "" {
			fmt.Println("用法: kb search <问题>")
			return
		}
		chunks, err := a.SearchKnowledge(ctx, query, 3)
		if err != nil {
			fmt.Println("错误:", err)
			return
		}
		if len(chunks) == 0 {
			fmt.Println("（无匹配结果）")
			return
		}
		for i, c := range chunks {
			fmt.Printf("[%d] score=%.3f source=%s\n%s\n", i+1, c.Score, c.Source, c.Content)
		}
	default:
		fmt.Println("用法: kb ingest <目录> | kb add <文本> | kb search <问题> | kb seed")
	}
}
