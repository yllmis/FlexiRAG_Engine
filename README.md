# FlexiRAG Engine

一个基于 Go 构建的多租户 RAG Agent 引擎，当前已具备可运行的 MVP v1：
- 支持长文本自动切片、向量化、持久化到 PostgreSQL（pgvector）
- 支持按 `agent_id` 隔离的知识检索与问答
- 支持 GLM（OpenAI 兼容模式）对话与 Embedding

## v1 功能范围

- 健康检查：`GET /ping`
- 知识摄入：`POST /api/v1/knowledge/ingest`
- 问答接口：`POST /api/v1/chat`
- 向量存储：`PGVectorStore`（PostgreSQL + pgvector）

## 目录结构（核心）

- `cmd/server`：HTTP 服务入口
- `internal/core/knowledge`：长文本切片与知识摄入编排
- `internal/core/agent_mgmt`：Agent 领域模型
- `internal/engine`：RAG 问答执行器
- `internal/infrastructure/llm`：GLM/OpenAI 客户端适配
- `internal/infrastructure/vector`：向量存储实现（Mock / pgvector）
- `pkg/textsplit`：通用文本切片工具

## 运行环境

- Go `1.23+`
- Docker / Docker Compose（用于 PostgreSQL + pgvector）
- GLM API Key（当前读取环境变量名：`OPENAI_API_KEY`）

## 快速开始

### 1. 启动 PostgreSQL（pgvector）

```bash
docker compose up -d
```

默认连接信息（见 `docker-compose.yml`）：
- Host: `localhost`
- Port: `5432`
- User: `root`
- Password: `12345`
- DB: `flexirag_db`

### 2. 设置 API Key

```bash
export OPENAI_API_KEY="你的GLM_API_KEY"
```

### 3. 启动服务

```bash
go run ./cmd/server/main.go
```

## API 示例

### 健康检查

```bash
curl -s http://127.0.0.1:8080/ping
```

### 长文本摄入（自动切片）

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/knowledge/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": 1,
    "text": "这里放一篇较长的文本内容...",
    "chunk_size": 300,
    "overlap": 40
  }'
```

### 问答

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"agent_id":1,"query":"四六级报名时间是什么时候？"}'
```

## 常见问题

### `password authentication failed for user "root"`

这通常不是代码问题，而是数据库密码与容器初始化状态不一致：
- 确认服务实际连接参数与 `docker-compose.yml` 一致
- 如果你曾改过账号密码，旧数据卷可能仍保留旧凭据

可重建（会清空数据库数据）：

```bash
docker compose down -v
docker compose up -d
```

## 当前限制（MVP）

- `cmd/server/main.go` 里数据库 DSN 为硬编码，建议下一步改为环境变量
- 尚未引入完整鉴权、限流、审计日志
- 尚未接入生产级向量索引参数调优（如 HNSW/IVFFlat）

## 下一步建议

- 配置化：将 DSN、模型名、超时等统一放入配置文件/环境变量
- 观测性：增加结构化日志与请求追踪
- 数据治理：完善 `agent_id` + `id` 复合唯一约束与迁移脚本
