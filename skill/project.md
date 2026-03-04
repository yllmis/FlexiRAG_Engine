# 🚀 FlexiRAG Engine: 基于 Go 的高性能多租户 AI Agent 平台

## 📌 项目简介 (Project Overview)
本项目是一个基于 Go 语言构建的**企业级、多租户 AI Agent SaaS 平台**。
与传统的“单体 AI 聊天机器人”不同，本平台允许不同租户（用户）动态创建和配置专属的 AI 智能体（Agent）。每个 Agent 拥有独立的“系统人设（System Prompt）”和“私有向量知识库（RAG）”，实现数据的绝对物理隔离与高度定制化。

**核心应用场景：** 企业内部知识库问答、高校教务系统智能客服、电商垂直领域自动售前客服等。

## 🛠️ 技术栈选型 (Tech Stack)
* **后端开发语言：** Go (Golang) 1.21+ 
    * *选型理由：利用 Goroutine 实现轻量级高并发处理，极其适合 I/O 密集型的 LLM API 调用场景。*
* **Web 框架：** Gin
* **大模型驱动层：** OpenAI API (采用兼容模式设计，支持平滑无缝切换至 DeepSeek, Qwen 等模型)
* **数据库引擎 (规划中)：** PostgreSQL + GORM
* **向量检索 (RAG)：** 内存级 Mock 检索引擎 -> 演进为 Milvus / pgvector
* **缓存与限流 (规划中)：** Redis

## 🏗️ 核心架构设计 (Architecture Design)
本项目严格遵循**整洁架构 (Clean Architecture)** 与 **依赖倒置原则 (DIP)**，保证了核心业务逻辑的高内聚与低耦合。

1. **接口层 (Ports)：** 在 `internal/core` 定义了 `LLMProvider` 和 `VectorStore` 等标准接口。
2. **引擎层 (Engine)：** `internal/engine/executor.go` 负责核心 RAG 调度引擎，处理“向量化 -> 检索 -> 组装 -> 生成”的闭环流转。
3. **适配器层 (Adapters)：** 在 `pkg/llm` 和 `pkg/vector` 中实现具体的第三方 SDK 接入（如 OpenAI 客户端封装），随时可热插拔。
4. **防御性编程：** 全链路透传 `context.Context`，实现大模型长耗时调用的超时控制与协程安全退出。

## ✨ 核心功能模块 (Key Features)

### 1. 多租户 Agent 引擎 (Multi-Tenant Agent System)
* 支持动态创建 Agent，自由配置 `System Prompt`。
* 通过 `AgentID` 字段实现严格的知识库物理隔离，防止跨租户数据越权（Data Leakage）。

### 2. RAG 知识检索增强流 (RAG Pipeline)
* **高效向量化：** 利用大模型 Embedding 接口的批处理（Batching）能力，优化长文本写入性能。
* **精准检索：** 预留 `float32` 类型的高维向量检索能力，平衡内存开销与精度。
* **防注入组装：** 使用 `<context>` 标签对检索内容进行严格边界划分，抵御 Prompt Injection 攻击，降低大模型幻觉（Hallucination）。

### 3. 可扩展工具链 (Plugin System - 规划中)
* 预留 `internal/tools` 模块，未来支持接入自定义爬虫（如抓取教务处通知）、天气查询等外部 API，赋予 Agent 与现实世界交互的能力。

## 💡 工程亮点 (Engineering Highlights)
* **【内存优化】** 向量数据强制使用 `float32` 替代默认的 `float64`，在 1536 维数据下节省 50% 的内存与存储开销。
* **【性能优化】** 在组装高维 RAG 上下文时，使用 `strings.Builder` 替代原生 `+` 拼接，规避字符串不可变性带来的内存频繁重分配与 GC 抖动。
* **【解耦设计】** 采用工厂方法与策略模式封装 LLM 客户端，业务代码零侵入即可实现跨模型厂商切换。
* **【鲁棒性保障】** 通过 `len(vectors) == 0` 的边界条件判空与全局 `Context` 级联取消，防止 API 异常导致的 `panic` 及 Goroutine 内存泄漏。

## 🗺️ 项目演进路线图 (Roadmap)
- [x] **Phase 1: 核心引擎骨架 (MVP)**
  - [x] 定义核心接口契约 (Interface)。
  - [x] 实现 RAG 执行器 (`Executor`) 的调度编排。
  - [x] 接入 OpenAI SDK 实现 Chat 与 Embed。
- [x] **Phase 2: 本地闭环运行**
  - [x] 实现基于内存的 Mock 向量数据库 (Cosine Similarity 计算)。
  - [x] 开发 Gin HTTP 路由，对外暴露 `/api/v1/chat` 接口。
  - [x] 使用 Postman 跑通首次完整问答。
- [ ] **Phase 3: 持久化与生产级改造**
  - [ ] 接入 PostgreSQL 持久化 Agent 配置。
  - [ ] 实现后台离线文本切片（Chunking）与批量向量化入库逻辑。
- [ ] **Phase 4: 高并发与微服务能力**
  - [ ] 基于 Redis 实现令牌桶（Token Bucket）API 频率限流。
  - [ ] 引入 Worker 并发池处理后台耗时任务。