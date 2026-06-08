// 第 3 步演示程序：最小可运行的 DeepSeek 调用示例。
//
// 运行前（二选一）：
//   方式 A：在项目根目录创建 .env 文件（推荐，程序会自动加载）
//   方式 B：export DEEPSEEK_API_KEY=sk-你的key
//
// 运行（必须在项目根目录 golangtest/ 下执行）：
//   go run ./cmd/llm-demo
//
// 程序做了什么？
//   1. 检查环境变量
//   2. 创建 LLM Client
//   3. 发一条 system + user 消息
//   4. 打印模型回复和 token 用量
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhjing1019/goAiAgent/internal/llm"
)

func main() {
	// ---------- 1. 创建 Client ----------
	// NewClientFromEnv 会先加载 .env，再读取 DEEPSEEK_API_KEY 等配置
	client, err := llm.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// ---------- 2. 构造请求并调用 ----------
	// context.Background() 表示「没有超时限制的默认上下文」
	// 后面 Agent 里会改成带超时的 context.WithTimeout
	resp, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{
			// system：设定 AI 行为（Agent 的系统提示词）
			llm.NewSystemMessage("你是一个 Go Agent 助手，回答简洁。"),
			// user：用户问题
			llm.NewUserMessage("用一句话解释什么是 Agent？"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// ---------- 3. 解析并打印结果 ----------
	// AssistantMessage() 是 types.go 里的 helper，取 choices[0].message
	msg := resp.AssistantMessage()
	if msg == nil {
		log.Fatal("模型没有返回内容")
	}

	fmt.Println("模型回复:", msg.Content)

	// Usage 可能为 nil（某些错误响应没有 usage）
	if resp.Usage != nil {
		fmt.Printf("Token 用量: prompt=%d completion=%d total=%d\n",
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens,
		)
	}
}
