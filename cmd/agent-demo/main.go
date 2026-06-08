// 第 4–5 步演示：带工具调用的 Agent + MySQL 多轮记忆
//
// 运行（在项目根目录）：
//
//	go run ./cmd/agent-demo
//
// 命令：
//   - exit / quit  退出
//   - reset / new  清空记忆，下次对话创建新会话
//   - sessions     列出最近会话（需 MYSQL_DSN）
//   - load <id>    从数据库恢复会话（需 MYSQL_DSN）
//
// 示例问题：
//   - 现在几点？
//   - 帮我算 123 + 456
//   - 帮我算 6 × 7
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
	"github.com/zhjing1019/goAiAgent/internal/store/mysql"
)

func main() {
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

	ag, err := agent.New(agent.Config{
		Client: client,
		Tools:  agent.DefaultRegistry(),
		Store:  sessionStore,
		SystemPrompt: `你是一个 Go Agent 助手。
- 需要当前时间时，调用 get_current_time
- 需要计算两数之和时，调用 add_numbers
- 需要计算两数之积时，调用 multiply_numbers
- 拿到工具结果后，用自然语言回答用户`,
		MaxSteps: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🤖 Agent 已启动（输入 exit 退出）")
	fmt.Println("可用工具: get_current_time, add_numbers, multiply_numbers")
	if sessionStore != nil {
		fmt.Println("会话命令: sessions | load <id> | reset")
	}
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
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
			if sessionStore == nil {
				fmt.Println("未启用 MySQL，无法列出会话")
				continue
			}
			list, err := ag.ListSessions(ctx, 10)
			if err != nil {
				fmt.Println("错误:", err)
				continue
			}
			if len(list) == 0 {
				fmt.Println("（暂无会话）")
				continue
			}
			for _, s := range list {
				fmt.Printf("  %s  %s  %s\n", s.ID[:8]+"...", s.Title, s.UpdatedAt.Format("2006-01-02 15:04"))
			}
			continue
		}
		if strings.HasPrefix(input, "load ") {
			if sessionStore == nil {
				fmt.Println("未启用 MySQL，无法加载会话")
				continue
			}
			id := strings.TrimSpace(strings.TrimPrefix(input, "load "))
			if id == "" {
				fmt.Println("用法: load <session_id>")
				continue
			}
			if err := ag.LoadSession(ctx, id); err != nil {
				fmt.Println("错误:", err)
				continue
			}
			fmt.Printf("已加载会话 %s，共 %d 条历史消息\n", id[:8]+"...", len(ag.Messages()))
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
