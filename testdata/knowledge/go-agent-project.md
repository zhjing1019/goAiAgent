# Go AI Agent 项目架构说明

本项目是一个用 Go 语言实现的 AI Agent 演示系统，集成了大模型对话、工具调用、MySQL 会话持久化和 Milvus RAG 知识库。适合作为学习 Agent 开发和 RAG 系统的参考项目。

## 技术栈

- **语言**：Go 1.26
- **LLM 框架**：langchaingo（OpenAI 兼容接口）
- **大模型**：DeepSeek（deepseek-chat）
- **Embedding**：OpenAI 兼容服务（如 SiliconFlow BGE）
- **向量库**：Milvus 2.6（通过 langchaingo milvus vectorstore）
- **数据库**：MySQL 8（会话和消息持久化）
- **配置管理**：.env 多环境隔离（development / staging / production）

## 项目结构

```
cmd/agent-demo/     主程序入口，交互式命令行
cmd/kb-ingest/      知识库批量导入工具
internal/agent/       Agent 核心：循环、工具、记忆
internal/llm/         LLM 客户端封装
internal/rag/         RAG：切分、Embedding、KnowledgeBase 接口
internal/rag/milvus/  Milvus 知识库实现
internal/store/mysql/ MySQL 会话存储
internal/config/      环境变量配置加载
testdata/knowledge/   示例知识文档
```

## Agent 工作流程

1. 用户输入问题
2. Agent 将历史消息 + 系统提示 + 用户输入发送给 LLM
3. LLM 返回文本回答，或返回 tool_calls（工具调用请求）
4. 如果有 tool_calls，Agent 执行对应工具，将结果追加到消息历史
5. 再次调用 LLM，直到返回最终文本回答
6. 整个对话自动保存到 MySQL

可用工具：
- get_current_time：获取当前时间
- add_numbers / multiply_numbers：数学计算
- search_knowledge：从 Milvus 检索知识（RAG 启用时）
- add_knowledge：写入知识库（RAG 启用时）

## 环境隔离设计

通过 APP_ENV 切换环境：
- development → .env + .env.development
- staging → .env + .env.staging
- production → 仅系统环境变量

每个环境使用独立的 MySQL 数据库和 Milvus Collection，避免数据污染。

## Milvus 集成细节

连接配置通过环境变量注入：
- MILVUS_ADDR：Milvus 服务地址，本地为 127.0.0.1:19530
- MILVUS_COLLECTION：Collection 名称，开发环境建议 go_agent_kb_dev
- MILVUS_TOKEN：Zilliz Cloud 需要，本地留空

写入时自动切分：rag.SplitText(content, 500) 将长文本按 500 字符切片，每片作为独立 Entity 写入 Milvus，metadata 包含 source 和 chunk 编号。

检索时使用 SimilaritySearch，返回 TopK 结果及其相似度分数。

## 启动方式

```bash
# 确保 Milvus 运行
docker start milvus-standalone

# 导入知识文档
make kb-ingest

# 启动 Agent
make run-dev
```

## 扩展方向

- 支持 PDF/Word 文档解析入库
- 升级为递归切分 + overlap
- 增加 Reranker 二次排序
- 接入 Hybrid Search（向量 + 关键词）
- 添加 RAG 评估脚本
- 部署为 HTTP API 服务

本项目的设计哲学是「最小可用 + 清晰分层」：每个模块职责单一，通过接口解耦，便于逐步替换和升级各个组件。
