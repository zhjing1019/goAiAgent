# Go Todo — 命令行 + Web API 待办清单

用 Go 编写的待办事项小项目，同时提供**命令行交互**和 **HTTP API** 两种使用方式，数据持久化到本地 `todos.json`。

> 适合 Go 入门学习：struct、slice、函数、指针、JSON、HTTP 等知识点均有覆盖。

## 功能

- 添加、查看、标记完成、删除待办
- 命令行模式（CLI）
- Web API 模式（默认）
- 本地 JSON 文件自动保存 / 加载

## 环境要求

- Go 1.26+

## 快速开始

```bash
git clone git@github.com:zhjing1019/goAiAgent.git
cd goAiAgent

# Web 模式（默认，监听 :8080）
go run .

# 命令行模式
go run . -cli

# 指定端口
go run . -addr :9090
```

## 项目结构

```
.
├── main.go      # 程序入口，选择 CLI / Web 模式
├── cli.go       # 命令行交互
├── server.go    # HTTP API
├── todo.go      # 数据结构与业务逻辑
├── todos.json   # 持久化数据（运行后生成/更新）
└── go.mod
```

## Web API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/todos` | 获取所有待办 |
| GET | `/todos/stats` | 获取统计 `{ "total": N, "pending": M }` |
| POST | `/todos` | 添加待办，body: `{"title":"内容"}` |
| PATCH | `/todos/{id}/complete` | 标记完成 |
| DELETE | `/todos/{id}` | 删除待办 |

### 示例

```bash
# 查看列表
curl http://localhost:8080/todos

# 添加
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"学习 Go"}'

# 统计
curl http://localhost:8080/todos/stats

# 标记完成
curl -X PATCH http://localhost:8080/todos/1/complete

# 删除
curl -X DELETE http://localhost:8080/todos/1
```

## CLI 菜单

```
1. 添加待办
2. 查看列表
3. 标记完成
4. 删除待办
5. 保存并退出
```

## 编译

```bash
go build -o todo-app .
./todo-app
```

## License

MIT
