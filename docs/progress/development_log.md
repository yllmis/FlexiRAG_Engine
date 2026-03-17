# 开发进度记录

本文档用于记录项目阶段性变更，便于新开窗口后快速了解当前状态与开发进度。

## 2026-03-16

- 已同步 README 与安全方案文档口径：明确当前已完成 P0 安全基线（静态 Token 鉴权、单机限流、异步审计三件套）。
- README 已补充 `security` 配置示例与环境变量说明：`ADMIN_TOKEN`、`RATE_LIMIT_PER_MINUTE`、`AUDIT_QUEUE_SIZE`。
- README 已新增“为什么 P0 先用静态 Token”说明，阐明分阶段策略（先最小防护，再升级 IAM）。
- 安全基线方案文档已新增“为什么 P0 采用静态 Token”章节，明确其过渡性质与后续升级路径（JWT/API Key）。
- 前端鉴权来源已收敛为仅 `VITE_ADMIN_TOKEN`：移除 `localStorage.admin_token` 覆盖逻辑，避免联调时因多来源冲突导致 401。
- 前端请求层已新增一次性诊断日志：启动后首个请求会提示是否成功读取 `VITE_ADMIN_TOKEN`（不输出敏感值，仅输出长度）。
- 限流实现已改为官方令牌桶库 `golang.org/x/time/rate`：保持按 key 独立限流与现有接口不变，移除自实现令牌补充逻辑。

## 2026-03-15

- 已完成安全基线模块 P0 代码落地：新增静态 Token 鉴权、中间件限流、异步审计（有界队列 + 降级 + 可观测计数）。
- 路由已收敛为“读接口公开、写接口保护”：`POST/PUT/chat/ingest` 需 Bearer Token 且受限流控制。
- 配置层新增 `security` 配置段，支持 `ADMIN_TOKEN`、`RATE_LIMIT_PER_MINUTE`、`AUDIT_QUEUE_SIZE`。
- 新增审计仓储 `audit_logs` 表自动迁移，以及异步审计写入 Worker。
- 前端 API 客户端已支持注入 `Authorization: Bearer <token>`（读取 `localStorage.admin_token` 或 `VITE_ADMIN_TOKEN`）。
- 补充单测：鉴权、限流、异步审计、配置覆盖等关键路径。

## 2026-03-13

- 已新增本地私有配置文件 `configs/app.local.yaml`，并在 `.gitignore` 中加入忽略规则，避免 API Key 等敏感信息误入库。
- 已按运行诉求将统一成功业务码回切为 `code = 200`：后端 `respondSuccess` 使用 `http.StatusOK` 作为 code，前端拦截器同步恢复 `code === 200` 判定。
- 已完成后端配置化改造：新增 `configs/app.yaml` 与 `internal/config` 配置加载模块，移除 `cmd/server/main.go` 中 DSN 与端口硬编码。
- 启动入口支持通过 `APP_CONFIG_PATH` 指定外部配置文件路径，便于多环境部署。
- 数据库连接改为通过 `internal/infrastructure/database.NewPostgresDB` 注入配置，支持 `sslmode/timezone` 配置项。
- 新增配置加载单测 `internal/config/app_config_test.go`，覆盖默认值、环境变量覆盖与 API Key 必填校验。
- 已确认并删除废弃页面文件：`web/src/views/agents/AgentCreateView.vue`、`web/src/views/rag/IngestView.vue`，并清理遗留的 `web/src/views/rag/IngestView.vue.js`。

## 2026-03-09

- 后端接口统一为 `code/msg/data` 响应结构，当前成功返回 `code = 200`。
- Agent 更新接口已从 `PATCH /api/v1/agents/:id/system-prompt` 收敛为 `PUT /api/v1/agents/:id`，支持更新 `name` 与 `system_prompt`。
- 新增 Agent 管理能力：创建 Agent、查询花名册、更新 Agent。
- 新增前端控制台 `web/`（Vue 3 + Vite + TypeScript + Tailwind）。
- 前端页面已覆盖：Agent 花名册、创建 Agent、编辑 Agent、知识入库、问答。
- 问答与知识入库页面已改为按 Agent 名称下拉选择，不再手输未创建的 `agent_id`。
- `.gitignore` 已补充 `web/node_modules/`、`web/dist/`、`web/.vite/`、`web/*.tsbuildinfo`、`web/src/**/*.js`，避免前端构建产物进入版本库。
- 前端页面结构已升级为两大主场景：C 端沉浸式问答台与 B 端 Agent 管理后台。
- 问答台新增左侧 Agent 花名册、按 Agent 维度保留本地对话上下文、气泡式消息区与底部输入栏。
- 管理后台新增 Agent 卡片墙、右侧抽屉式创建/编辑表单，以及支持拖拽文本文件的知识库面板。
- 智能体卡片墙已统一卡片底部控件高度，并将“设为入库对象”改为“同步到知识面板”，明确该动作仅更新前端本地选择状态，不新增后端接口。
- 智能体卡片墙底部两个控件已进一步统一为等宽等高，卡片内视觉对齐更稳定。
- 智能体卡片已改为纵向弹性布局，提示词摘要区域自动撑开，确保底部按钮区在不同卡片中的相对位置一致。
- 对话台已删除 Agent 的系统提示词和编号等补充展示，侧栏与头部仅保留 Agent 名称，减少干扰。

## 当前前端状态

- 可通过 `cd web && npm run dev` 启动本地前端。
- 可通过 `cd web && npm run build` 完成构建验证。
- 当前界面已切换为双场景控制台，兼顾高频问答与后台配置；后续仍可继续补充真实会话持久化与文件上传接口。
