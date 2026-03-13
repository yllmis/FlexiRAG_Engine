# FlexiRAG Engine

一个基于 Go 构建的多租户 RAG Agent 引擎，当前已具备可运行的 v1.1：
- 支持长文本自动切片、向量化、持久化到 PostgreSQL（pgvector）
- 支持按 `agent_id` 隔离的知识检索与问答
- 支持 GLM（OpenAI 兼容模式）对话与 Embedding
- 支持 Agent 创建、花名册查询、系统提示词更新
- 支持 `web/` 前端控制台（Vue 3 + Vite + TypeScript + Tailwind）

## v1.1 功能范围

- 健康检查：`GET /ping`
- 创建 Agent：`POST /api/v1/agents`
- Agent 花名册：`GET /api/v1/agents`
- 更新 Agent：`PUT /api/v1/agents/:id`
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
- `web`：前端控制台工程（Vite）

## 运行环境

- Go `1.23+`
- Docker / Docker Compose（用于 PostgreSQL + pgvector）
- GLM API Key（当前读取环境变量名：`OPENAI_API_KEY`）

## 配置文件

服务默认从 `configs/app.yaml` 读取运行配置：

```yaml
server:
  port: 8080

database:
  host: 127.0.0.1
  port: 5432
  user: root
  password: 12345
  dbname: flexirag_db
  sslmode: disable
  timezone: Asia/Shanghai

llm:
  provider: glm
  api_key: ""
  base_url: https://open.bigmodel.cn/api/paas/v4/
  chat_model: glm-4-flash
  embed_model: embedding-3
```

环境变量可覆盖配置文件：

- `APP_CONFIG_PATH`（可选，默认 `configs/app.yaml`）
- `SERVER_PORT`
- `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` `DB_SSLMODE` `DB_TIMEZONE`
- `OPENAI_API_KEY` `LLM_PROVIDER` `LLM_BASE_URL` `LLM_CHAT_MODEL` `LLM_EMBED_MODEL`

## 快速开始

### 1. 启动 PostgreSQL（pgvector）

```bash
docker compose up -d postgres
docker compose exec -T postgres psql -U root -d flexirag_db -c "CREATE EXTENSION IF NOT EXISTS vector;"
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

### 4. 启动前端（可选）

```bash
cd web
npm install
npm run dev
```

访问：`http://127.0.0.1:3000`

说明：前端通过 `web/vite.config.ts` 代理 `/api` 与 `/ping` 到后端 `http://127.0.0.1:8080`，本地联调无需后端开启 CORS。

## 统一响应体

当前后端接口统一返回结构：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

- 成功：`code = 200`
- 失败：`code = 4xx/5xx`，错误信息在 `msg`
- 业务数据统一放在 `data`

## API 示例

### 健康检查

```bash
curl -s http://127.0.0.1:8080/ping
```

示例响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "message": "pong"
  }
}
```

### 创建 Agent

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "教务小助手",
    "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
  }'
```

示例响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "agent_id": 1,
    "name": "教务小助手",
    "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
  }
}
```

### 查询 Agent 花名册

```bash
curl -s http://127.0.0.1:8080/api/v1/agents
```

### 更新 Agent

```bash
curl -s -X PUT http://127.0.0.1:8080/api/v1/agents/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"教务升级版助手","system_prompt":"你是资深教务顾问，回答需简洁准确。"}'
```

说明：`name` 与 `system_prompt` 都是可选字段，但至少传一个。

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

注意：`chat` 和 `knowledge/ingest` 现在都要求显式传入 `agent_id`，不再自动兜底到默认 Agent。

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

- 尚未引入完整鉴权、限流、审计日志
- 尚未接入生产级向量索引参数调优（如 HNSW/IVFFlat）

## 下一步建议

- 配置化：将 DSN、模型名、超时等统一放入配置文件/环境变量
- 观测性：增加结构化日志与请求追踪
- 数据治理：完善 `agent_id` + `id` 复合唯一约束与迁移脚本
