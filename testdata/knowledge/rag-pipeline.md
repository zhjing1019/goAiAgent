# RAG 检索增强生成完整流程

RAG（Retrieval-Augmented Generation）是当前企业 AI 应用最主流的架构之一。它让大模型能够基于私有知识库回答问题，而不是仅依赖训练时的记忆。

## 为什么需要 RAG

大模型的局限：
- 训练数据有截止日期，不知道最新信息
- 不了解企业内部文档、流程、产品细节
- 容易「幻觉」——编造看似合理但实际错误的内容
- 无法引用来源，难以审计

RAG 的解决思路：先从知识库检索相关片段，再把这些片段作为上下文交给 LLM 生成答案。这样答案有据可查，且知识库可以持续更新。

## RAG 流水线五步

### 1. 文档采集（Ingestion）

数据来源包括：Markdown、PDF、Word、网页、数据库记录、API 文档、客服对话历史。

采集时注意：
- 保留文档元数据（标题、作者、更新时间、分类）
- 统一编码为 UTF-8
- 去除无意义页眉页脚、广告内容
- 大文件需要解析器（如 PyMuPDF、Unstructured）

本项目的 testdata/knowledge/ 目录存放 Markdown 示例文档，通过 kb-ingest 工具批量入库。

### 2. 文本切分（Chunking）

长文档不能直接 Embedding，需要切成适当大小的片段。

**固定长度切分**（本项目当前方案）：
- 按 500 字符切分，实现简单，适合入门
- 缺点：可能在句子中间切断，丢失上下文

**递归字符切分**：
- 优先按段落、句子、标点切分，再按字数兜底
- langchain 的 RecursiveCharacterTextSplitter 是业界标准

**语义切分**：
- 用 Embedding 相似度判断语义边界
- 效果最好，但计算成本较高

**最佳实践：**
- Chunk 大小 300-800 字（中文）
- Overlap 50-100 字
- 每个 Chunk 附带 metadata：source、chunk_index、total_chunks、title

### 3. 向量化（Embedding）

Embedding 模型将文本映射到高维向量空间，语义相近的文本向量距离更近。

常用模型：
- 中文：BAAI/bge-large-zh-v1.5（1024维）、m3e-base
- 英文：text-embedding-3-small（1536维）、text-embedding-3-large
- 多语言：multilingual-e5-large

选择 Embedding 服务的考量：
- SiliconFlow：国内访问快，支持 BGE 系列，有免费额度
- OpenAI：质量稳定，需海外网络
- 本地部署：数据不出内网，需 GPU

**关键原则：入库和检索必须使用同一个 Embedding 模型，维度必须一致。**

### 4. 向量存储（Milvus）

将 Embedding 向量和原文、元数据一起写入 Milvus Collection。

写入模式：
- 实时写入：用户通过 add_knowledge 工具即时入库
- 批量写入：kb-ingest 工具扫描目录批量导入
- 增量更新：文档变更时删除旧 chunk 再写入新 chunk

### 5. 检索与生成（Retrieval + Generation）

用户提问时的流程：
1. 将问题 Embedding 为 Query Vector
2. Milvus TopK 相似度搜索（通常 K=3-10）
3. （可选）Reranker 精排
4. 将检索结果拼入 Prompt
5. LLM 基于上下文生成答案

Prompt 模板示例：
```
你是企业知识助手。根据以下参考资料回答问题。
如果资料中没有相关信息，请明确说「我不知道」。

参考资料：
{retrieved_chunks}

用户问题：{user_query}
```

## 评估 RAG 质量

不能凭感觉判断 RAG 好坏，需要量化评估：

- **召回率（Recall）**：正确文档片段是否被检索到
- **精确率（Precision）**：检索结果中有多少是真正相关的
- **答案忠实度（Faithfulness）**：LLM 回答是否基于检索内容
- **端到端准确率**：最终答案是否正确

工具：RAGAS 框架、自建评估集（50-100 个问答对）。

## 本项目中的 RAG 实现

Go Agent 项目通过 KnowledgeBase 接口解耦向量库：

- `internal/rag/knowledge.go` — 接口定义（Add / Search）
- `internal/rag/milvus/kb.go` — Milvus 实现
- `internal/rag/chunk.go` — 文本切分
- `internal/rag/embedder.go` — Embedding 客户端
- `internal/agent/rag_tools.go` — Agent 工具（search_knowledge / add_knowledge）

环境变量：
- MILVUS_ADDR=127.0.0.1:19530
- MILVUS_COLLECTION=go_agent_kb_dev
- EMBEDDING_API_KEY + EMBEDDING_BASE_URL + EMBEDDING_MODEL

命令行操作：
- `kb ingest testdata/knowledge` — 批量导入文档
- `kb search 企业级 Milvus 怎么部署` — 手动检索测试
- Agent 对话中自动调用 search_knowledge 工具
