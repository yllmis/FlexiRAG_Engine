# 安全基线模块任务拆解与实施方案（v1）

## 1. 模块目标

在不破坏现有 RAG 主链路（创建 Agent、知识入库、问答）的前提下，补齐最小安全基线：

- 接口鉴权（Authentication）
- 基础限流（Rate Limit）
- 审计记录（Audit Log）

交付后达到：

- 未授权写接口不可访问
- 高频滥用请求可被限制
- 关键操作可追溯到操作者与请求上下文

---

## 2. 范围与边界

### 2.1 本期范围（P0）

- 写接口强制鉴权：
  - `POST /api/v1/agents`
  - `PUT /api/v1/agents/:id`
  - `POST /api/v1/knowledge/ingest`
  - `POST /api/v1/chat`
- 限流粒度：按 token（无 token 时按 IP）
- 审计事件：创建 Agent、更新 Agent、知识入库、问答调用

### 2.2 暂不纳入（P1/P2）

- RBAC 细粒度权限模型
- 分布式限流（Redis）
- 完整 SIEM 对接

### 2.3 为什么 P0 采用静态 Token

P0 目标是先完成最小安全闭环，优先解决“匿名可写”的高风险问题，因此选择静态 Token 作为过渡方案。

- 快速落地：不依赖登录、注册、刷新、密钥管理等整套账号体系。
- 主链路低扰动：只在中间件层增加校验，不改业务 Handler 核心流程。
- 可验证性强：可以稳定复现并验收 `401/429` 和审计链路。
- 可替换性好：后续升级 JWT/API Key 时，Router/Handler 与审计链路可保持不变，主要替换 AuthService 实现。

结论：静态 Token 不是终态，而是“先安全、后治理”的阶段化实现。

---

## 3. 架构与分层落地顺序

遵循仓库约定：`Router -> Handler -> Repository/Port` 变更时先补 Port，再补基础设施实现，再接 Handler 与 Router。

1. `core/ports.go`：新增 Auth/Audit/RateLimiter Port
2. `internal/infrastructure`：补 Token 校验、限流器、审计仓储实现
3. `internal/api/v1/handler.go`：接入用户上下文与审计打点
4. `internal/api/v1/router.go`：挂中间件并收敛写接口保护策略

---

## 4. 任务拆解清单（可直接开工）

### 4.1 P0-1 鉴权中间件

- 新增 `AuthService` Port（校验 token，返回主体信息）
- 新增 `middlewares/auth.go`：
  - 从 `Authorization: Bearer <token>` 读取 token
  - 校验失败返回 `401`
  - 校验成功写入 `gin.Context`（如 `subject_id`）
- Router 对写接口挂载鉴权中间件

交付物：

- `internal/core/ports.go`（新增 Port）
- `internal/infrastructure/auth/*`（实现）
- `internal/api/v1/middlewares/auth.go`
- `internal/api/v1/router.go`（挂载）

验收：

- 无 token 调用写接口返回 `401`
- 合法 token 调用写接口返回 `200`

### 4.1.1 P0-1A 发牌逻辑（鉴权闭环补齐）

本期采用最小 MVP：静态配置 Token 发放，不引入登录系统。

- 在配置中新增管理员 Token（建议环境变量优先）：
  - `ADMIN_TOKEN`（或配置文件字段 `security.admin_token`）
- 前端本地联调时使用该 Token 注入 `Authorization` 头
- `AuthService` 仅负责校验 Bearer Token 是否等于配置值，并产出固定主体（如 `subject_id=admin`）
- 登录接口（如 `POST /api/v1/login`）与动态签发 API Key 属于下一阶段

交付物：

- `internal/config/*`（新增 `security.admin_token` 配置项）
- `internal/infrastructure/auth/static_token.go`
- `web/src/api/http.ts`（支持注入静态 Token）

验收：

- 配置了 `ADMIN_TOKEN` 后，带正确 Bearer Token 可访问写接口
- Token 错误或缺失时返回 `401`
- 文档明确声明本期为“静态 Token 发放”，动态发牌未纳入

### 4.2 P0-2 基础限流

- 新增 `RateLimiter` Port
- 实现本地内存令牌桶（按 token/IP 维度）
- 在写接口中间件链路加入限流

交付物：

- `internal/core/ports.go`（新增限流 Port）
- `internal/infrastructure/ratelimit/in_memory.go`
- `internal/api/v1/middlewares/ratelimit.go`

验收：

- 超过阈值返回 `429`
- 正常请求不受影响

### 4.3 P0-3 审计日志

- 新增 `AuditRepository` Port
- 异步审计实现必须满足“三件套”强约束：
  - 有界队列：使用固定容量 Channel，禁止无限起 Goroutine
  - 降级策略：队列满时丢弃并记录告警，不阻塞主请求
  - 可观测性：暴露投递总数、丢弃数、写入失败数等指标或日志
- 设计最小审计字段：
  - `event_type`
  - `subject_id`
  - `resource_type`
  - `resource_id`
  - `request_id`
  - `status`
  - `created_at`
- 在 Handler 成功/失败路径打审计
- 审计写入采用异步模式：
  - Handler 内组装审计事件后，投递到审计 Channel
  - 后台 Worker Goroutine 批量或逐条写库
  - 当 Channel 满时采用降级策略（丢弃并打告警日志，不能阻塞主请求）

交付物：

- `internal/core/ports.go`（新增 Audit Port）
- `internal/infrastructure/repository/audit_db.go`
- `internal/infrastructure/audit/async_writer.go`
- `internal/api/v1/handler.go`（审计打点）

验收：

- 关键写操作可查审计记录
- 失败请求也可记录失败状态
- 主链路不因审计写库阻塞而显著增加时延
- 三件套生效：
  - 队列容量可配置且超过容量时不阻塞请求
  - 丢弃事件与失败写入有可追踪日志/指标

### 4.4 P0-4 前端最小适配

- `web/src/api/http.ts` 在 `401/429` 时给出可读提示
- 允许注入 `Authorization` 头（先支持静态 token）

交付物：

- `web/src/api/http.ts`

验收：

- 前端对 `401/429` 有明确提示

---

## 5. 接口与数据契约

### 5.1 响应语义

保持当前统一约定：

- 成功：`code = 200`
- 失败：`code = http status`

### 5.2 中间件上下文字段

- `subject_id`：鉴权主体
- `request_id`：请求唯一标识（可由网关透传或服务生成）

---

## 6. 测试策略

### 6.1 单元测试

- `auth`：token 缺失、token 无效、token 有效
- `ratelimit`：阈值内放行、超阈值拦截
- `audit`：成功路径记录、失败路径记录
- `audit-async`：
  - Channel 正常消费
  - Channel 满时降级行为正确（不阻塞请求）
  - Worker 异常恢复与告警日志行为正确
  - 队列容量边界条件正确（容量为 1、容量满、连续突发）

### 6.2 接口测试

- 写接口无 token 返回 `401`
- 写接口高频压测触发 `429`
- 关键写接口可在审计表查到记录

### 6.3 统一验收命令

- `go test ./...`

---

## 7. 风险与回滚

### 7.1 风险

- 鉴权默认全开导致联调中断
- 限流阈值配置不当导致误杀
- 审计写入失败影响主链路时延
- 异步队列积压导致审计事件丢失

### 7.2 回滚策略

- 鉴权开关可配置（灰度开启）
- 限流开关可配置（逐步收紧阈值）
- 审计失败降级为告警日志，不阻塞主请求
- 审计异步队列可配置长度与丢弃策略，必要时切换为同步兜底模式（仅限短期排障）

---

## 8. 里程碑与工期建议

- M1（0.5 天）：Port 设计与路由保护点确认
- M2（1 天）：鉴权中间件 + 单测
- M3（1 天）：限流中间件 + 单测
- M4（1 天）：审计仓储 + Handler 打点 + 单测
- M5（0.5 天）：联调与回归（前端提示、文档更新）

总计建议：约 4 天

---

## 9. 模块评审清单（完成后执行）

- 是否遵循 `Port -> Infra -> Handler -> Router`
- 是否覆盖 `401/429/200` 关键路径
- 是否有可追溯审计记录
- 是否补齐对应 `_test.go`
- 是否执行并通过 `go test ./...`

---

## 10. 模块考核题（完成后用于回顾）

1. 为什么鉴权中间件应优先放在 Router 层而非 Handler 内部？
2. 本地内存限流与 Redis 分布式限流的核心差异是什么？
3. 审计日志为什么要记录失败事件，而不只记录成功事件？
4. 在当前项目中，如何保证安全改造不破坏 `code=200` 成功语义？
5. 如果审计库短暂不可用，主请求链路应如何降级？
