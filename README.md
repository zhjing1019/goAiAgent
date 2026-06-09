# Go AI Agent

Go 语言实现的 AI Agent 演示项目：多轮对话、工具调用、MySQL 会话持久化、Milvus RAG 知识库。

## 环境要求

- Go 1.26+（推荐用 GVM 管理，见 `make gvm-install`）
- Docker Desktop（本地 Milvus）
- MySQL 8（可选，会话持久化）
- SiliconFlow 等 OpenAI 兼容 Embedding 服务（RAG 必填）

## 快速开始

```bash
# 1. 初始化环境文件
make env-init

# 2. 编辑 .env.development，填入 DEEPSEEK_API_KEY、MYSQL_DSN、EMBEDDING_API_KEY 等

# 3. 启动 Agent（开发环境）
make run-dev
```

## 环境配置

配置文件加载顺序：`.env` → `.env.{APP_ENV}` → `.env.{APP_ENV}.local`

| 变量 | 说明 |
|------|------|
| `DEEPSEEK_API_KEY` | 大模型 API Key（必填） |
| `MYSQL_DSN` | MySQL 连接串（可选） |
| `MILVUS_ADDR` | Milvus 地址，如 `127.0.0.1:19530` |
| `MILVUS_COLLECTION` | Collection 名，开发环境建议 `go_agent_kb_dev` |
| `EMBEDDING_API_KEY` | Embedding API Key（RAG 必填） |
| `EMBEDDING_BASE_URL` | 如 `https://api.siliconflow.cn/v1` |
| `EMBEDDING_MODEL` | 如 `BAAI/bge-large-zh-v1.5` |

---

## 常用命令速查

### 项目 / Agent

```bash
make run-dev                              # 启动 Agent（development）
make run-staging                          # 启动 Agent（staging）
make build                                # 编译 agent-demo
make test                                 # 运行测试
make env-init                             # 初始化 .env.development
make deps-init                            # 下载 Go 依赖
```

### Milvus 知识库（RAG）

```bash
# 批量导入 testdata/knowledge/ 下的 .md 文档（切分 + 向量化 + 写入 Milvus）
make kb-ingest

# 手动检索验证
make kb-search QUERY='企业级 Milvus 怎么部署'
make kb-search QUERY='RAG 切分策略'
```

Agent 启动后还可输入：

```text
kb ingest testdata/knowledge    # 批量导入指定目录
kb add <文本>                   # 手动写入一条知识
kb search <问题>              # 手动检索
kb seed                         # 写入 3 条示例知识
```

### Docker / Milvus 运维

```bash
# 确认 Docker 已启动（菜单栏有鲸鱼图标）
docker ps

# 首次启动 Milvus（仅需执行一次，见下方「Milvus 首次部署」）
docker start milvus-standalone

# 电脑重启后恢复 Milvus
docker start milvus-standalone

# 确认 Milvus 端口
lsof -i :19530

# 查看容器状态
docker ps --filter name=milvus-standalone

# 查看日志（排查问题时）
docker logs milvus-standalone --tail 50

# 停止 Milvus
docker stop milvus-standalone
```

### Milvus 首次部署（Intel Mac）

若还没有 `milvus-standalone` 容器，在项目外任意目录执行：

```bash
mkdir -p ~/milvus-standalone/volumes/milvus

cat > ~/milvus-standalone/embedEtcd.yaml << 'EOF'
listen-client-urls: http://0.0.0.0:2379
advertise-client-urls: http://0.0.0.0:2379
quota-backend-bytes: 4294967296
auto-compaction-mode: revision
auto-compaction-retention: '1000'
EOF

touch ~/milvus-standalone/user.yaml

docker run -d \
  --name milvus-standalone \
  --security-opt seccomp:unconfined \
  -e DEPLOY_MODE=STANDALONE \
  -e ETCD_USE_EMBED=true \
  -e ETCD_DATA_DIR=/var/lib/milvus/etcd \
  -e ETCD_CONFIG_PATH=/milvus/configs/embedEtcd.yaml \
  -e COMMON_STORAGETYPE=local \
  -v ~/milvus-standalone/volumes/milvus:/var/lib/milvus \
  -v ~/milvus-standalone/embedEtcd.yaml:/milvus/configs/embedEtcd.yaml \
  -v ~/milvus-standalone/user.yaml:/milvus/configs/user.yaml \
  -p 19530:19530 \
  -p 9091:9091 \
  -p 2379:2379 \
  milvusdb/milvus:v2.6.0 \
  milvus run standalone
```

> 注意：Milvus 2.6 单容器启动必须设置 `DEPLOY_MODE=STANDALONE` 和内嵌 etcd，否则容器会秒退。

### Attu 可视化

1. 打开 Attu 桌面版
2. 连接地址：`127.0.0.1:19530`，数据库 `default`，Token 留空
3. 点左侧 **Collections** 图标（不是首页），找到 `go_agent_kb_dev`
4. 进入后查看 **Data**（数据）或 **Vector Search**（搜索）

Milvus WebUI（系统监控）：http://127.0.0.1:9091/webui

### GVM（Go 版本管理）

```bash
make gvm-install                          # 安装 GVM + 项目 Go 版本
source scripts/gvm-use.sh               # 激活 GVM 环境
gvm list                                  # 查看已安装 Go 版本
```

---

## 每日开发流程

```bash
# 1. 启动 Docker Desktop
# 2. 启动 Milvus
docker start milvus-standalone

# 3. 导入/更新知识库（文档有变更时）
make kb-ingest

# 4. 启动 Agent
make run-dev
```

---

## 常见问题

| 现象 | 处理 |
|------|------|
| `docker: command not found` | 打开 Docker Desktop |
| `Cannot connect to docker daemon` | 等 Docker 完全启动后再执行命令 |
| Attu 首页显示 Collection 数量为 0 | 点左侧 Collections 图标进入列表 |
| `metric type invalid` | 已在代码中设置 `COSINE`，删除旧 Collection 后重新 `make kb-ingest` |
| `rate limit exceeded` | 稍等后重试，`kb-ingest` 已内置重试 |
| `kb-ingest` 报 Embedding 错误 | 检查 `.env.development` 中 `EMBEDDING_API_KEY` |
| 电脑重启后 RAG 不可用 | 执行 `docker start milvus-standalone` |

---

## 项目结构

```
cmd/agent-demo/          Agent 交互入口
cmd/kb-ingest/           批量导入文档到 Milvus
cmd/kb-search/           命令行检索验证
internal/agent/          Agent 循环与工具
internal/rag/            切分、Embedding、知识库接口
internal/rag/milvus/     Milvus 实现
internal/store/mysql/    MySQL 会话存储
internal/config/         环境变量配置
testdata/knowledge/      示例知识文档（Milvus 入门、RAG、企业实践等）
```

## License

MIT
