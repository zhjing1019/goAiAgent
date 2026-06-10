// Package app 把 LLM、工具、MySQL、Milvus 装配成可运行的 Agent 服务（第 7 步）。
//
// 整体架构（对应 LangGraph 的「图式执行」，Go 里用手写循环实现）：
//
//	┌─────────────┐
//	│  用户输入    │
//	└──────┬──────┘
//	       ▼
//	┌─────────────┐     ┌──────────────┐
//	│   Agent     │────▶│  DeepSeek    │  LLM 推理
//	│   Run()     │◀────│  (langchaingo)│
//	└──────┬──────┘     └──────────────┘
//	       │
//	       ├── tool_calls? ──▶ Registry.Execute()
//	       │                      ├─ get_current_time
//	       │                      ├─ add_numbers / multiply_numbers
//	       │                      ├─ search_knowledge ──▶ Milvus RAG
//	       │                      └─ add_knowledge    ──▶ Milvus RAG
//	       │
//	       ├── appendMessage ──▶ MySQL SessionStore（可选）
//	       │
//	       └── 纯文本回复 ──▶ 返回用户
//
// 本包职责：上面所有组件的「组装 / 生命周期 / 对外 API」，不包含具体工具实现。
package app
