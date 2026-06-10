package app

import (
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/config"
)

// Status 启动时各子系统是否就绪（不含密钥）。
type Status struct {
	Env          string
	MySQLEnabled bool
	RAGEnabled   bool
}

// PrintStartup 打印环境与子系统状态。
func (s Status) PrintStartup() {
	fmt.Printf("🌍 %s\n", config.EnvSummary())
	if s.MySQLEnabled {
		fmt.Println("✅ MySQL 已连接，对话将自动保存")
	} else {
		fmt.Println("ℹ️  未配置 MYSQL_DSN，使用内存记忆（重启后丢失）")
	}
	if s.RAGEnabled {
		fmt.Println("✅ Milvus 知识库已连接（search_knowledge / add_knowledge 已启用）")
	} else {
		fmt.Println("ℹ️  未配置 MILVUS_ADDR + EMBEDDING_API_KEY，RAG 未启用")
	}
}

// PrintHelp 打印 CLI 可用命令。
func (s Status) PrintHelp() {
	fmt.Println("可用工具: get_current_time, add_numbers, multiply_numbers")
	if s.RAGEnabled {
		fmt.Println("RAG 工具: search_knowledge, add_knowledge")
		fmt.Println("知识库命令: kb ingest <目录> | kb add <文本> | kb search <问题> | kb seed")
	}
	if s.MySQLEnabled {
		fmt.Println("会话命令: sessions | load <id> | reset")
	}
}
