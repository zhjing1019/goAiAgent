# 企业级 Milvus 实践指南

在企业项目中引入 Milvus，不仅要会「跑起来」，更要考虑可用性、安全性、成本控制和可运维性。本章梳理从 PoC 到生产的完整路径。

## 第一阶段：PoC 验证（1-2 周）

目标：验证 RAG 方案在你的业务数据上是否有效。

- 使用 Standalone 或 Zilliz Cloud 免费 tier
- 准备 50-200 篇代表性文档
- 选定 Embedding 模型（中文推荐 BGE-large-zh、M3E；英文推荐 text-embedding-3-small）
- 实现基础切分（500-800 字/块，带 overlap 更好）
- 评估指标：召回率、答案准确率、响应延迟

PoC 阶段不要过度优化索引参数，FLAT 或默认 HNSW 足够。重点是验证「检索到的内容能否支撑 LLM 给出正确答案」。

## 第二阶段：开发集成（2-4 周）

目标：将 RAG 嵌入现有应用架构。

**数据流设计：**
```
文档源 → 解析(PDF/Word/HTML) → 清洗 → 切分 → Embedding → Milvus 写入
用户提问 → Query Embedding → Milvus 检索 → Rerank(可选) → Prompt 组装 → LLM
```

**Collection 设计原则：**
- 按业务域拆分 Collection，而非全部塞进一个表
- 标量字段预留：source、doc_id、chunk_id、title、category、updated_at
- 向量维度与 Embedding 模型严格一致
- 生产/预发/开发使用不同 Collection 名（如 go_agent_kb_dev / _staging / _prod）

**SDK 集成：**
- Go 服务通过 KnowledgeBase 接口抽象向量库，便于切换 Milvus / Pinecone / Qdrant
- 写入和检索使用独立连接池，设置合理超时
- 批量写入优于逐条写入（本项目 kb-ingest 工具即为此设计）

## 第三阶段：生产部署（4-8 周）

**部署选型：**

| 方案 | 适用场景 | 运维成本 |
|------|----------|----------|
| Zilliz Cloud | 快速上线、免运维 | 低 |
| K8s + Milvus Operator | 自建、需完全掌控 | 高 |
| Docker Compose | 中小规模单机 | 中 |

**高可用要点：**
- etcd 集群至少 3 节点
- MinIO/S3 对象存储独立部署
- QueryNode 多副本，Proxy 前置负载均衡
- 定期备份：milvus-backup 工具支持全量和增量

**监控告警：**
- Prometheus + Grafana 采集 Milvus 指标
- 关注：查询延迟 P99、索引构建时间、内存使用率、Segment 数量
- Attu v2.6+ 内置 Prometheus Dashboard
- 慢查询日志定期分析

**安全：**
- 开启 RBAC 认证（root/Milvus 为默认账号，上线后必须改密码）
- 生产环境启用 TLS
- Collection 级别权限隔离
- API Key / Token 通过环境变量注入，禁止硬编码

## 第四阶段：持续优化

**切分策略升级：**
- 固定字数切分（当前项目方案）→ 按段落/标题切分 → 语义切分（Semantic Chunking）
- 增加 chunk overlap（50-100 字）减少边界信息丢失
- 表格、代码块单独处理

**检索优化：**
- Hybrid Search：向量 + BM25 关键词
- Reranker 模型二次排序（如 bge-reranker）
- Query 改写：将用户口语化问题转为检索友好表述
- 多路召回：不同 Collection 或不同切分粒度并行检索后合并

**索引调优：**
- 数据量 < 100 万：HNSW, M=16, efConstruction=256
- 数据量 > 100 万：考虑 IVF + PQ 压缩
- 定期执行 compact 和 index 重建

## 学习路径推荐

1. **官方文档**：https://milvus.io/docs — 从 Quickstart 到 Architecture 通读一遍
2. **动手实验**：本地 Standalone + Attu + 本项目 kb-ingest/kb search
3. **源码阅读**：关注 Collection、Segment、QueryNode 相关设计
4. **Zilliz 博客**：企业案例和最佳实践
5. **进阶**：Milvus Bootcamp（GitHub）、Zilliz 开源社区

企业级 Milvus 的核心不是「会用 Attu 点按钮」，而是理解数据全生命周期：采集 → 切分 → 向量化 → 存储 → 检索 → 评估 → 迭代。
