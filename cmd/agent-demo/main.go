// 第 4–6 步演示：Agent + MySQL 记忆 + Milvus RAG
//
// 运行（在项目根目录）：
//
//	go run ./cmd/agent-demo
//
// 命令：
//   - exit / quit       退出
//   - reset / new       清空对话记忆
//   - sessions          列出 MySQL 会话（需 MYSQL_DSN）
//   - load <id>         恢复 MySQL 会话
//   - kb add <文本>     手动写入知识库（需 Milvus + Embedding）
//   - kb search <问题>  手动检索知识库
//   - kb seed           写入示例知识（便于测试 RAG）
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zhjing1019/goAiAgent/internal/agent"
	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/rag"
	ragmilvus "github.com/zhjing1019/goAiAgent/internal/rag/milvus"
	"github.com/zhjing1019/goAiAgent/internal/store/mysql"
)

func main() {
	fmt.Printf("🌍 %s\n", config.EnvSummary())

	client, err := llm.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	var sessionStore *mysql.Store
	mysqlCfg, err := config.LoadMySQL()
	if err != nil {
		log.Fatal(err)
	}
	if mysqlCfg.Enabled() {
		if err := mysql.EnsureDatabase(mysqlCfg.DSN); err != nil {
			log.Fatalf("MySQL 初始化失败: %v\n请检查 .env 中的 MYSQL_DSN", err)
		}
		sessionStore, err = mysql.Open(mysqlCfg.DSN)
		if err != nil {
			log.Fatalf("MySQL 连接失败: %v\n请检查 .env 中的 MYSQL_DSN", err)
		}
		defer sessionStore.Close()

		ctx := context.Background()
		if err := sessionStore.Migrate(ctx); err != nil {
			log.Fatalf("MySQL 建表失败: %v", err)
		}
		fmt.Println("✅ MySQL 已连接，对话将自动保存")
	} else {
		fmt.Println("ℹ️  未配置 MYSQL_DSN，使用内存记忆（重启后丢失）")
	}

	ctx := context.Background()
	kb, err := ragmilvus.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("RAG 初始化失败: %v", err)
	}
	if kb != nil {
		fmt.Println("✅ Milvus 知识库已连接（search_knowledge / add_knowledge 已启用）")
	} else {
		fmt.Println("ℹ️  未配置 MILVUS_ADDR + EMBEDDING_API_KEY，RAG 未启用")
	}

	systemPrompt := buildSystemPrompt(kb != nil)
	ag, err := agent.New(agent.Config{
		Client:       client,
		Tools:        agent.DefaultRegistry(kb),
		Store:        sessionStore,
		SystemPrompt: systemPrompt,
		MaxSteps:     10,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🤖 Agent 已启动（输入 exit 退出）")
	printHelp(kb != nil, sessionStore != nil)
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
			ag.Reset()
			fmt.Println("（对话已清空，下次将创建新会话）")
			continue
		}
		if input == "sessions" {
			handleSessions(ctx, ag, sessionStore)
			continue
		}
		if strings.HasPrefix(input, "load ") {
			handleLoad(ctx, ag, sessionStore, input)
			continue
		}
		if strings.HasPrefix(input, "kb ") {
			handleKB(ctx, kb, input)
			continue
		}

		answer, err := ag.Run(ctx, input)
		if err != nil {
			fmt.Println("错误:", err)
			continue
		}
		fmt.Println("Agent:", answer)
		if sid := ag.SessionID(); sid != "" && sessionStore != nil {
			fmt.Printf("（会话 ID: %s）\n", sid)
		}
	}
}

func buildSystemPrompt(ragEnabled bool) string {
	prompt := `你是一个 Go Agent 助手。
- 需要当前时间时，调用 get_current_time
- 需要计算两数之和时，调用 add_numbers
- 需要计算两数之积时，调用 multiply_numbers
- 拿到工具结果后，用自然语言回答用户`
	if ragEnabled {
		prompt += `
- 当问题涉及文档、政策、产品说明、已入库知识时，先调用 search_knowledge 检索
- 用户明确要求「记住/保存到知识库」时，调用 add_knowledge
- 回答时结合检索到的内容，不知道就说不知道，不要编造`
	}
	return prompt
}

func printHelp(ragEnabled, mysqlEnabled bool) {
	fmt.Println("可用工具: get_current_time, add_numbers, multiply_numbers")
	if ragEnabled {
		fmt.Println("RAG 工具: search_knowledge, add_knowledge")
		fmt.Println("知识库命令: kb add <文本> | kb search <问题> | kb seed")
	}
	if mysqlEnabled {
		fmt.Println("会话命令: sessions | load <id> | reset")
	}
}

func handleSessions(ctx context.Context, ag *agent.Agent, store *mysql.Store) {
	if store == nil {
		fmt.Println("未启用 MySQL，无法列出会话")
		return
	}
	list, err := ag.ListSessions(ctx, 10)
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

func handleLoad(ctx context.Context, ag *agent.Agent, store *mysql.Store, input string) {
	if store == nil {
		fmt.Println("未启用 MySQL，无法加载会话")
		return
	}
	id := strings.TrimSpace(strings.TrimPrefix(input, "load "))
	if id == "" {
		fmt.Println("用法: load <session_id>")
		return
	}
	if err := ag.LoadSession(ctx, id); err != nil {
		fmt.Println("错误:", err)
		return
	}
	fmt.Printf("已加载会话 %s，共 %d 条历史消息\n", id[:8]+"...", len(ag.Messages()))
}

func handleKB(ctx context.Context, kb rag.KnowledgeBase, input string) {
	if kb == nil {
		fmt.Println("RAG 未启用，请配置 MILVUS_ADDR 和 EMBEDDING_API_KEY")
		return
	}
	rest := strings.TrimSpace(strings.TrimPrefix(input, "kb "))
	switch {
	case rest == "seed":
		seedKB(ctx, kb)
	case strings.HasPrefix(rest, "add "):
		text := strings.TrimSpace(strings.TrimPrefix(rest, "add "))
		if text == "" {
			fmt.Println("用法: kb add <文本>")
			return
		}
		if err := kb.Add(ctx, text, "cli"); err != nil {
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
		chunks, err := kb.Search(ctx, query, 3)
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
		fmt.Println("用法: kb add <文本> | kb search <问题> | kb seed")
	}
}

func seedKB(ctx context.Context, kb rag.KnowledgeBase) {
	docs := []struct {
		content string
		source  string
	}{
		{"Go Agent 项目支持多轮对话、工具调用、MySQL 持久化和 Milvus RAG。", "项目简介"},
		{"DeepSeek 是 OpenAI 兼容的大模型 API，本项目通过 langchaingo 调用。", "模型说明"},
		{"Agent 循环：用户输入 → LLM → 有 tool_calls 就执行工具 → 再调 LLM → 直到返回文本。", "架构说明"},
	}
	for _, d := range docs {
		if err := kb.Add(ctx, d.content, d.source); err != nil {
			fmt.Println("写入失败:", err)
			return
		}
	}
	fmt.Println("✅ 已写入 3 条示例知识，可问：「这个项目有什么功能？」")
}
