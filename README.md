# FlexiRAG Engine

一个基于 Go 构建的多租户私有化 RAG Agent 引擎，当前版本已具备完整后端 API + 前端控制台闭环能力：
- 支持长文本自动切片（Chunking）与 Overlap 重叠分片
- 支持 Embedding 向量化并持久化到 PostgreSQL（pgvector）
- 支持按 `agent_id` 隔离的知识检索与问答
- 支持 Agent 创建、查询、更新、删除
- 支持静态 Token 鉴权、按主体/IP 限流、异步审计
- 支持 `web/` 前端控制台（Vue 3 + Vite + TypeScript + Tailwind）

## 功能清单

- 健康检查：`GET /ping`（公开）
- Agent 花名册：`GET /api/v1/agents`（公开）
- 创建 Agent：`POST /api/v1/agents`（需鉴权）
- 更新 Agent：`PUT /api/v1/agents/:id`（需鉴权）
- 删除 Agent：`DELETE /api/v1/agents/:id`（需鉴权）
- 知识摄入：`POST /api/v1/knowledge/ingest`（需鉴权）
- 问答接口：`POST /api/v1/chat`（需鉴权）

## 核心目录

- `cmd/server`：HTTP 服务入口
- `internal/api/v1`：路由、处理器、中间件（Auth/RateLimit/RequestID）
- `internal/core`：领域模型与 Port 接口定义
- `internal/engine`：RAG 问答执行引擎
- `internal/core/knowledge`：长文本切片与知识摄入编排
- `internal/infrastructure/llm`：GLM/OpenAI 兼容客户端
- `internal/infrastructure/vector`：向量存储实现（Mock/pgvector）
- `internal/infrastructure/ratelimit`：基于 `golang.org/x/time/rate` 的限流器
- `internal/infrastructure/audit`：异步审计写入
- `web`：前端控制台工程（Vite）

## 运行环境

- Go `1.25.0+`
- Node.js `18+`（前端开发）
- Docker / Docker Compose（用于 PostgreSQL + pgvector）
- GLM/OpenAI 兼容 API Key（环境变量：`OPENAI_API_KEY`）

## 配置说明

服务默认读取 `configs/app.yaml`，也可通过 `APP_CONFIG_PATH` 指向其他配置文件（如 `configs/app.local.yaml`）。

关键配置项：
- `security.admin_token`：后端静态鉴权 Token
- `security.rate_limit_per_minute`：每分钟限流额度（默认 60）
- `security.audit_queue_size`：异步审计队列大小（默认 1024）

环境变量可覆盖配置文件：
- `APP_CONFIG_PATH`（默认 `configs/app.yaml`）
- `SERVER_PORT`
- `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` `DB_SSLMODE` `DB_TIMEZONE`
- `OPENAI_API_KEY` `LLM_PROVIDER` `LLM_BASE_URL` `LLM_CHAT_MODEL` `LLM_EMBED_MODEL`
- `ADMIN_TOKEN` `RATE_LIMIT_PER_MINUTE` `AUDIT_QUEUE_SIZE`

## 快速开始

### 1. 启动 PostgreSQL（pgvector）

```bash
docker compose up -d postgres
docker compose exec -T postgres psql -U root -d flexirag_db -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### 2. 配置 LLM Key

在[配置文件](configs/app.yaml)中写入`api_key: ""`,可以自行调换模型，目前支持glm和openai

### 3. 启动后端

```bash
APP_CONFIG_PATH=configs/app.local.yaml go run ./cmd/server/main.go
```

### 4. 启动前端

```bash
cd web
npm install
cat > .env.local <<'EOF'
VITE_ADMIN_TOKEN=flexirag-secret-123
EOF
npm run dev
```

访问：`http://127.0.0.1:3000`

说明：前端通过 `web/vite.config.ts` 代理 `/api` 与 `/ping` 到后端 `http://127.0.0.1:8080`，本地联调无需后端开启 CORS。

## 统一响应格式

所有接口统一返回：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

- 成功：`code = 200`
- 失败：`code = 4xx/5xx`
- 业务数据：统一放在 `data`

## API 示例

### 健康检查（公开）

```bash
curl -s http://127.0.0.1:8080/ping
```

### 查询 Agent 花名册（公开）

```bash
curl -s http://127.0.0.1:8080/api/v1/agents
```

### 创建 Agent（鉴权）

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/agents \
  -H "Authorization: Bearer flexirag-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "教务小助手",
    "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
  }'
```

### 更新 Agent（鉴权）

```bash
curl -s -X PUT http://127.0.0.1:8080/api/v1/agents/1 \
  -H "Authorization: Bearer flexirag-secret-123" \
  -H "Content-Type: application/json" \
  -d '{"name":"教务升级版助手","system_prompt":"你是资深教务顾问，回答需简洁准确。"}'
```

说明：`name` 与 `system_prompt` 均为可选，但至少提供一个。

### 删除 Agent（鉴权）

```bash
curl -s -X DELETE http://127.0.0.1:8080/api/v1/agents/1 \
  -H "Authorization: Bearer flexirag-secret-123"
```

### 长文本摄入（鉴权）

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/knowledge/ingest \
  -H "Authorization: Bearer flexirag-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": 1,
    "text": "这里放一篇较长的文本内容...",
    "chunk_size": 300,
    "overlap": 40
  }'
```

### 问答（鉴权）

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/chat \
  -H "Authorization: Bearer flexirag-secret-123" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":1,"query":"四六级报名时间是什么时候？"}'
```

注意：`chat` 与 `knowledge/ingest` 都要求显式传入 `agent_id`。

## 安全与稳定性基线

当前采用 P0 安全基线：
- 鉴权：静态 Bearer Token（`security.admin_token` / `ADMIN_TOKEN`）
- 限流：基于 `golang.org/x/time/rate` 的单机内存限流（按 token/IP）
- 审计：异步写入（有界队列 + 队列满降级策略）

已知后续演进方向：
- 动态发牌与权限模型（JWT/API Key + RBAC）
- 分布式限流（Redis）
- 生产级审计接入（SIEM/告警平台）

## 常见问题

### 1) `401 未授权，请检查 Bearer Token`

检查以下两点：
- 后端 `security.admin_token` 与前端 `web/.env.local` 中 `VITE_ADMIN_TOKEN` 是否一致
- 修改 `.env.local` 后是否重启了 `npm run dev`

### 2) `password authentication failed for user "root"`

这通常是数据库容器初始化凭据与当前配置不一致：
- 确认连接参数与 `docker-compose.yml` 一致
- 若历史数据卷残留旧密码，可重建（会清空数据库）

```bash
docker compose down -v
docker compose up -d
```

## 测试

后端回归测试：

```bash
go test ./...
```
