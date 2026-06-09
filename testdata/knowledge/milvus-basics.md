# Milvus 向量数据库入门

Milvus 是一款开源的云原生向量数据库，专为海量向量数据的存储、索引和相似度检索而设计。在大模型时代，Milvus 是 RAG（检索增强生成）、推荐系统、图像搜索、语义搜索等场景的核心基础设施。

## 核心概念

**Collection（集合）** 类似于关系数据库中的表，是存储向量数据的基本单位。每个 Collection 有固定的 Schema，定义了字段名称、数据类型和维度。向量字段（如 embedding）必须指定维度，例如 1024 维的 BGE 模型输出就是 1024 维浮点数组。

**Entity（实体）** 是 Collection 中的一条记录，包含向量字段和标量字段。标量字段用于过滤，例如 source、category、created_at。向量字段用于相似度计算。

**Partition（分区）** 是 Collection 内的逻辑划分，适合按时间或业务线隔离数据。查询时可以指定分区，减少扫描范围。

**Index（索引）** 是加速向量检索的数据结构。常见索引类型包括 FLAT（暴力搜索，精度最高）、IVF_FLAT、IVF_SQ8、HNSW 等。HNSW 在大多数场景下兼顾了速度和召回率，是企业项目常用的选择。

**Segment** 是 Milvus 内部的数据组织单元，负责将数据持久化到对象存储。理解 Segment 有助于排查数据加载慢、内存占用高等问题。

## 架构组件

Milvus 支持 Standalone（单机）和 Cluster（分布式集群）两种部署模式。

Standalone 模式适合开发测试，所有组件（Proxy、QueryNode、DataNode、IndexNode、MixCoord、etcd、MinIO）运行在一个进程或容器中。你当前本地 Docker 启动的就是 Standalone 模式。

Cluster 模式将各组件拆分为独立服务，通过 Kubernetes 编排，支持水平扩展。生产环境通常使用 Cluster 模式，配合 Zilliz Cloud 托管服务可以进一步降低运维成本。

**Proxy** 是客户端请求的入口，负责路由和负载均衡。**QueryNode** 执行向量检索。**DataNode** 负责数据写入和 Compaction。**IndexNode** 负责构建向量索引。**Coord** 系列组件负责元数据管理和调度。

## 检索流程

一次典型的向量检索流程如下：

1. 用户输入自然语言问题
2. Embedding 模型将问题转为向量（Query Vector）
3. Milvus 在指定 Collection 中做 ANN（近似最近邻）搜索
4. 返回 TopK 最相似的向量及其关联文本
5. LLM 基于检索到的上下文生成回答

Milvus 支持混合检索：向量相似度 + 标量过滤。例如「在 source=hr-policy 的文档中，找与『年假』最相关的 5 段内容」。

## 与 Attu 的关系

Attu 是 Milvus 的图形化管理工具。通过 Attu 你可以：查看 Collection Schema、浏览数据、执行向量搜索、监控集群健康。本地开发时连接 127.0.0.1:19530 即可。

Milvus WebUI（端口 9091）侧重系统监控：节点状态、慢查询、Segment 信息。Attu 侧重数据操作，两者互补。

## 版本选择建议

Milvus 2.5+ 引入了内置 WebUI 和更多可观测性能力。Milvus 2.6 是当前稳定主线版本。SDK 版本应与 Server 版本尽量匹配，避免 API 兼容问题。Go 项目推荐使用 milvus-sdk-go/v2 或 langchaingo 的 milvus vectorstore 封装。
