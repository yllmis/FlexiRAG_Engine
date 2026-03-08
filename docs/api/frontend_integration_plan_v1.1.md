# 前后端分离对接方案（v1.1 修订版）

本文档为前后端分离联调契约文档，采用评审后的统一架构约束，前后端均以本文件为准。

## 1. 目标与范围

- 面向版本：`v1.1`
- 适用模式：前后端分离（前端独立部署，后端提供 HTTP API）
- 业务范围：
  - Agent 管理：创建、花名册、更新 Agent 信息（`name/system_prompt`）
  - RAG 能力：知识摄入、问答
  - 健康检查：服务可用性探测

## 2. 统一响应体规范（核心契约）

所有后端响应必须统一为：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

规范说明：
- `code = 0`：业务成功
- `code != 0`：业务失败
- `msg`：成功或错误描述（原来的 `error` 字段合并到 `msg`）
- `data`：业务数据载体（`answer`、`agent_id`、`agents` 等都放这里）

建议错误码（前后端统一约定）：
- `0`：成功
- `40001`：请求参数错误
- `40401`：资源不存在
- `50001`：服务内部错误

## 3. 接口总览

Base URL（本地后端）：`http://127.0.0.1:8080`

- `GET /ping`
- `POST /api/v1/agents`
- `GET /api/v1/agents`
- `PUT /api/v1/agents/:id`
- `POST /api/v1/knowledge/ingest`
- `POST /api/v1/chat`

说明：
- 评审后将更新 Agent 接口统一为 `PUT /api/v1/agents/:id`。
- `PUT` 请求体中 `name`、`system_prompt` 均为可选字段，至少传一个。

## 4. 前端工程建议目录（Vue 3 + Vite + TS + Tailwind）

```text
web/
  index.html
  package.json
  vite.config.ts
  tailwind.config.ts
  postcss.config.js
  src/
    main.ts
    App.vue
    router/
      index.ts
    api/
      http.ts                 # axios实例、请求/响应拦截、统一错误处理
      agents.ts               # Agent相关API
      rag.ts                  # chat/ingest相关API
    stores/
      agent.ts                # Pinia：当前Agent、花名册缓存
      ui.ts                   # Pinia：全局提示、加载状态
    types/
      common.ts               # 通用响应体类型
      agent.ts
      rag.ts
    views/
      agents/
        AgentListView.vue
        AgentCreateView.vue
        AgentEditView.vue
      rag/
        IngestView.vue
        ChatView.vue
    components/
      agents/
        AgentTable.vue
        AgentForm.vue
      common/
        BaseCard.vue
        EmptyState.vue
        LoadingMask.vue
    composables/
      useAgent.ts
      useRequest.ts
    styles/
      index.css
```

## 5. API 契约（统一响应体版）

### 5.1 健康检查

`GET /ping`

成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "pong"
  }
}
```

失败响应（示例）：

```json
{
  "code": 50001,
  "msg": "健康检查失败",
  "data": null
}
```

### 5.2 创建 Agent

`POST /api/v1/agents`

请求体：

```json
{
  "name": "教务小助手",
  "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
}
```

成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "agent_id": 1,
    "name": "教务小助手",
    "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
  }
}
```

失败响应（参数错误）：

```json
{
  "code": 40001,
  "msg": "参数错误，需要 name 和 system_prompt 字段",
  "data": null
}
```

### 5.3 Agent 花名册

`GET /api/v1/agents`

成功响应（字段统一为 `agent_id`）：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "agents": [
      {
        "agent_id": 1,
        "name": "教务小助手",
        "system_prompt": "你是一个严谨的教务助理，请仅依据上下文回答。"
      }
    ]
  }
}
```

失败响应（示例）：

```json
{
  "code": 50001,
  "msg": "查询 Agent 花名册失败",
  "data": null
}
```

### 5.4 更新 Agent（name/system_prompt）

`PUT /api/v1/agents/:id`

请求体（`name`、`system_prompt` 可选，至少传一个）：

```json
{
  "name": "新名称（可选）",
  "system_prompt": "新提示词（可选）"
}
```

成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "agent_id": 1,
    "name": "新名称（可选）",
    "system_prompt": "新提示词（可选）"
  }
}
```

失败响应（Agent 不存在）：

```json
{
  "code": 40401,
  "msg": "Agent 不存在",
  "data": null
}
```

### 5.5 知识摄入

`POST /api/v1/knowledge/ingest`

请求体：

```json
{
  "agent_id": 1,
  "text": "一段较长知识文本...",
  "chunk_size": 300,
  "overlap": 40
}
```

成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "知识入库成功，已持久化到 PostgreSQL",
    "agent_id": 1,
    "chunk_size": 300,
    "overlap": 40
  }
}
```

失败响应（参数错误示例）：

```json
{
  "code": 40001,
  "msg": "overlap 必须小于 chunk_size",
  "data": null
}
```

### 5.6 问答

`POST /api/v1/chat`

请求体：

```json
{
  "agent_id": 1,
  "query": "四六级报名时间是什么时候？"
}
```

成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "answer": "..."
  }
}
```

失败响应（示例）：

```json
{
  "code": 50001,
  "msg": "AI 思考失败，请稍后再试",
  "data": null
}
```

## 6. 前端 TypeScript 类型建议

```ts
export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

export interface AgentDTO {
  agent_id: number;
  name: string;
  system_prompt: string;
}

export interface CreateAgentReq {
  name: string;
  system_prompt: string;
}

export interface UpdateAgentReq {
  name?: string;
  system_prompt?: string;
}

export interface ListAgentsData {
  agents: AgentDTO[];
}

export interface ChatReq {
  agent_id: number;
  query: string;
}

export interface ChatData {
  answer: string;
}

export interface IngestReq {
  agent_id: number;
  text: string;
  chunk_size?: number;
  overlap?: number;
}
```

## 7. 前端错误处理约定

前端统一按 `code/msg/data` 处理：
- `code === 0`：业务成功，读取 `data`
- `code !== 0`：业务失败，展示 `msg`
- 网络错误或超时：统一提示 `网络异常，请稍后重试`

推荐封装（伪代码）：

```ts
if (resp.code !== 0) {
  throw new Error(resp.msg || "请求失败");
}
return resp.data;
```

## 8. 联调流程建议

1. 启动数据库容器并确保 pgvector 扩展启用。
2. 启动后端服务。
3. 启动前端 Vite 开发服务（通过代理访问后端）。
4. 先调 `GET /ping` 验证链路。
5. 调 `POST /api/v1/agents` 创建 Agent，保存 `agent_id`。
6. 调 `GET /api/v1/agents` 校验花名册展示。
7. 调 `PUT /api/v1/agents/:id` 更新 `name/system_prompt`。
8. 调 `POST /api/v1/knowledge/ingest` 和 `POST /api/v1/chat` 验证 RAG 全链路。

## 9. 本地跨域解决方案（必须执行）

后端不提供全局 CORS 中间件。前端必须通过 Vite 代理转发 `/api` 请求到后端，绕过浏览器跨域限制。

`vite.config.ts` 示例：

```ts
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true
      },
      "/ping": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true
      }
    }
  }
});
```

前端调用建议：
- 不要写死 `http://127.0.0.1:8080`
- 统一使用相对路径：`/api/v1/...`、`/ping`

## 10. 后续演进建议

- `GET /api/v1/agents` 增加分页参数：`page/page_size`
- 增加 `GET /api/v1/agents/:id` 详情接口
- 增加删除与禁用 Agent 能力
- 输出 OpenAPI 文档并自动生成前端 SDK
