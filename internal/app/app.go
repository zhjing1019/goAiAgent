package app

import (
	"context"
	"fmt"

	"github.com/zhjing1019/goAiAgent/internal/agent"
	"github.com/zhjing1019/goAiAgent/internal/config"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/rag"
	ragmilvus "github.com/zhjing1019/goAiAgent/internal/rag/milvus"
	"github.com/zhjing1019/goAiAgent/internal/redisx"
	"github.com/zhjing1019/goAiAgent/internal/store"
	"github.com/zhjing1019/goAiAgent/internal/store/cached"
	"github.com/zhjing1019/goAiAgent/internal/store/mysql"
)

// Config 应用级配置（可在 New 时覆盖默认值）。
type Config struct {
	SystemPrompt string
	MaxSteps     int
}

// App 单体 Agent 应用：统一装配 LLM + 工具 + MySQL + Milvus。
type App struct {
	agent    *agent.Agent   // CLI 默认 Agent（共享单会话）
	agentTpl agent.Config   // HTTP 按会话创建 Agent 的模板
	kb       rag.KnowledgeBase
	store       store.SessionStore
	mysql       *mysql.Store    // 非 nil 时 Close 会关闭连接
	redis       *redisx.Client  // 非 nil 时 Close 会关闭连接
	rateLimiter *redisx.RateLimiter
	status      Status
}

// NewFromEnv 从环境变量创建并装配完整 Agent 应用。
func NewFromEnv(ctx context.Context) (*App, error) {
	return New(ctx, Config{})
}

// New 装配 Agent 应用。Config 零值字段会使用合理默认。
func New(ctx context.Context, cfg Config) (*App, error) {
	client, err := llm.NewClientFromEnv()
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	var sessionStore store.SessionStore
	var mysqlStore *mysql.Store
	mysqlCfg, err := config.LoadMySQL()
	if err != nil {
		return nil, err
	}
	if mysqlCfg.Enabled() {
		fmt.Println("⏳ 正在连接 MySQL...")
		if err := mysql.EnsureDatabase(mysqlCfg.DSN); err != nil {
			return nil, fmt.Errorf("mysql bootstrap: %w", err)
		}
		mysqlStore, err = mysql.Open(mysqlCfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("mysql connect: %w", err)
		}
		if err := mysqlStore.Migrate(ctx); err != nil {
			_ = mysqlStore.Close()
			return nil, fmt.Errorf("mysql migrate: %w", err)
		}
		sessionStore = mysqlStore
	}

	// ---------- Redis 装配（可选）----------
	// 配置了 REDIS_ADDR 才启用；连不上会降级，不阻断启动。
	var redisClient *redisx.Client
	var rateLimiter *redisx.RateLimiter
	redisCfg, err := config.LoadRedis()
	if err != nil {
		if mysqlStore != nil {
			_ = mysqlStore.Close()
		}
		return nil, err
	}
	if redisCfg.Enabled() {
		fmt.Println("⏳ 正在连接 Redis...")
		redisClient, err = redisx.Open(ctx, redisCfg)
		if err != nil {
			fmt.Printf("⚠️  Redis 连接失败，已跳过缓存与限流: %v\n", err)
			fmt.Println("   启动 Redis: docker start redis  或  docker run -d --name redis -p 6379:6379 redis:7")
		} else {
			// 有 MySQL 时：用 cached.Store 包装，LoadSession 先走 Redis
			if sessionStore != nil {
				sessionStore = cached.New(sessionStore, redisClient.RDB(), redisCfg.SessionCacheTTL)
			}
			// REDIS_RATE_LIMIT > 0 时：HTTP 中间件对 /api/chat 限流
			if redisCfg.RateLimitPerMin > 0 {
				rateLimiter = redisx.NewRateLimiter(redisClient.RDB(), redisCfg.RateLimitPerMin)
			}
		}
	}

	milvusCfg, _ := config.LoadMilvus()
	var kb rag.KnowledgeBase
	if milvusCfg.Enabled() {
		fmt.Println("⏳ 正在连接 Milvus（最多等待 15 秒）...")
		kb, err = ragmilvus.OpenFromEnv(ctx)
		if err != nil {
			fmt.Printf("⚠️  Milvus 连接失败，RAG 未启用: %v\n", err)
			fmt.Println("   启动: docker start milvus-standalone")
			kb = nil
		}
	}

	ragEnabled := kb != nil
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = BuildSystemPrompt(ragEnabled)
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}

	tools := agent.DefaultRegistry(kb)
	agentCfg := agent.Config{
		Client:       client,
		Tools:        tools,
		Store:        sessionStore,
		SystemPrompt: cfg.SystemPrompt,
		MaxSteps:     cfg.MaxSteps,
	}
	ag, err := agent.New(agentCfg)
	if err != nil {
		if redisClient != nil {
			_ = redisClient.Close()
		}
		if mysqlStore != nil {
			_ = mysqlStore.Close()
		}
		return nil, fmt.Errorf("agent: %w", err)
	}

	return &App{
		agent:       ag,
		agentTpl:    agentCfg,
		kb:          kb,
		store:       sessionStore,
		mysql:       mysqlStore,
		redis:       redisClient,
		rateLimiter: rateLimiter,
		status: Status{
			Env:                 config.AppEnv(),
			MySQLEnabled:        mysqlStore != nil,
			RAGConfigured:       milvusCfg.Enabled(),
			RAGEnabled:          ragEnabled,
			RedisConfigured:     redisCfg.Enabled(),
			RedisEnabled:        redisClient != nil,
			SessionCacheEnabled: redisClient != nil && mysqlStore != nil,
			RateLimitEnabled:    rateLimiter != nil,
		},
	}, nil
}

// Close 释放数据库连接等资源。
func (a *App) Close() error {
	var err error
	if a.redis != nil {
		err = a.redis.Close()
	}
	if a.mysql != nil {
		if cerr := a.mysql.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

// RateLimiter 返回 HTTP 限流器（未配置 Redis 或 REDIS_RATE_LIMIT=0 时为 nil）。
func (a *App) RateLimiter() *redisx.RateLimiter {
	return a.rateLimiter
}

// Status 返回各子系统启用状态。
func (a *App) Status() Status {
	return a.status
}

// Run 处理用户输入，返回 Agent 最终回复。
func (a *App) Run(ctx context.Context, input string) (string, error) {
	return a.agent.Run(ctx, input)
}

// Reset 清空当前对话记忆。
func (a *App) Reset() {
	a.agent.Reset()
}

// SessionID 当前会话 ID。
func (a *App) SessionID() string {
	return a.agent.SessionID()
}

// LoadSession 从 MySQL 恢复历史会话。
func (a *App) LoadSession(ctx context.Context, sessionID string) error {
	return a.agent.LoadSession(ctx, sessionID)
}

// ListSessions 列出最近会话。
func (a *App) ListSessions(ctx context.Context, limit int) ([]store.Session, error) {
	return a.agent.ListSessions(ctx, limit)
}

// MessageCount 当前会话消息条数。
func (a *App) MessageCount() int {
	return len(a.agent.Messages())
}
