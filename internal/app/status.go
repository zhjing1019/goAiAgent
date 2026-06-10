package app

import (
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/config"
)

// Status 启动时各子系统是否就绪（不含密钥）。
type Status struct {
	Env                 string
	MySQLEnabled        bool
	RAGEnabled          bool
	RedisConfigured     bool // REDIS_ADDR 已配置
	RedisEnabled        bool // 实际连接成功
	SessionCacheEnabled bool // Redis + MySQL 会话缓存
	RateLimitEnabled    bool
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
	if s.RedisEnabled {
		fmt.Println("✅ Redis 已连接")
		if s.SessionCacheEnabled {
			fmt.Println("   └─ 会话消息缓存已启用（加速 LoadSession）")
		}
		if s.RateLimitEnabled {
			fmt.Println("   └─ HTTP /api/chat 限流已启用")
		}
	} else if s.RedisConfigured {
		fmt.Println("⚠️  REDIS_ADDR 已配置但连接失败，缓存与限流未启用")
		fmt.Println("   启动: docker run -d --name redis -p 6379:6379 redis:7")
	} else {
		fmt.Println("ℹ️  未配置 REDIS_ADDR，跳过缓存与限流")
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
