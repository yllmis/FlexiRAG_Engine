# FlexiRAG Engine

## 顶层目录用途

- `cmd/`：程序入口。
  - `cmd/server`：API 主服务进程。
  - `cmd/worker`：后台任务进程（如爬虫任务）。
- `configs/`：配置文件目录（YAML/环境配置）。
- `internal/`：私有业务代码（不对外暴露）。
  - `internal/api/`：HTTP API 层（版本与中间件）。
  - `internal/core/`：核心业务逻辑层。
  - `internal/engine/`：Agent 思考/规划/循环等执行引擎。
  - `internal/tools/`：工具与插件（如爬虫、计算器）。
  - `internal/data/`：数据访问层（数据库与缓存）。
  - `internal/model/`：数据模型定义。
  - `internal/pkg/`：内部通用工具（如 LLM、向量库封装）。

> 当前仅初始化目录结构与占位文件，不包含 `.go` 源码。
