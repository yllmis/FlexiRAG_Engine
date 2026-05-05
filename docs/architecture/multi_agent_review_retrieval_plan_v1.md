# 多 Agent 审核与检索编排方案（v1）

本文档用于规划 `FlexiRAG Engine` 的下一阶段演进方向：在保留当前单 Agent RAG 主链路可用性的前提下，引入多 Agent 审核能力与更完整的检索编排能力，让项目更突出 Agent 技术力、策略编排能力与可讲述性。

---

## 1. 方案目标

本期目标不是把项目做成重型生产系统，而是做成一个：

- 能体现 `Agent 协作` 的 AI 应用项目
- 能体现 `RAG 检索增强` 的技术理解
- 能体现 `可配置策略编排` 的系统设计能力
- 能用于简历展示、面试讲解和后续迭代的中等复杂度项目

本期完成后，项目应具备以下可展示能力：

- 主回答 Agent 与审核 Agent 的协作链路
- 检索前查询理解与查询改写
- 多路检索策略编排（Dense / Sparse / Multi-query）
- 融合排序与上下文压缩
- 可配置开关，支持个人模式与多人模式切换

---

## 2. 当前项目现状

当前后端已经具备以下基础能力：

- Agent 创建、查询、更新、删除
- 长文本切片与向量入库
- 基于 `agent_id` 的向量检索问答
- 静态 Token 鉴权、基础限流、异步审计
- Vue 3 管理台与问答台

当前主链路可抽象为：

```text
用户问题
  -> Handler
  -> AgentEngine
  -> Query Embedding
  -> Vector Search(topK=3)
  -> 拼接上下文
  -> 调用 LLM 生成答案
```

当前最明显的能力缺口：

- 只有单路向量检索，缺少检索策略编排
- 没有查询重写，用户问题质量差时检索效果弱
- 没有输入审核、输出审核、知识入库审核
- 没有结果重排、去重、融合与压缩
- 没有把“Agent 角色分工”显式表达出来

---

## 3. 总体设计方向

建议把系统升级为“多阶段 Agent-RAG 管线”：

```text
用户请求
  -> 输入审核 Agent（可选）
  -> 查询理解 Agent
       - 查询分类
       - 查询改写
       - 多查询扩展
  -> 检索编排器
       - Dense Retrieval
       - Sparse Retrieval(BM25 / PostgreSQL FTS)
       - HyDE Retrieval（后续可选）
  -> 融合排序器
       - RRF
       - 去重
       - TopK 裁剪
  -> 上下文压缩器
       - 关键句提取
       - 无关段落过滤
       - 文档融合摘要
  -> 主回答 Agent
  -> 输出审核 Agent（可选）
  -> 返回答案 + 引用 + 审核结果 + 策略元数据
```

该设计的核心思想：

- 将“问答”拆成多个职责清晰的阶段
- 将“检索”从单次向量搜索升级为策略编排
- 将“审核”从外围功能升级为可配置 Agent 能力
- 将“回答结果”从黑盒文本升级为可解释输出

---

## 4. 角色划分

### 4.1 主回答 Agent

职责：

- 基于最终上下文生成回答
- 遵守 Agent 的 `system_prompt`
- 尽量引用知识库事实，不脱离上下文发挥

建议保留当前 `AgentEngine` 的核心职责，但将其前置依赖从“单一向量检索结果”升级为“编排后的上下文结果”。

### 4.2 输入审核 Agent

职责：

- 判断用户输入是否包含不当、敏感、越权、攻击性内容
- 判断是否需要拒答、降级或打标

适用场景：

- 多人共用知识库
- 课堂/社群/企业内部试用

个人私有化模式可关闭，避免影响体验与成本。

### 4.3 查询理解 Agent

职责：

- 识别用户问题意图
- 将口语化问题改写为更适合检索的表达
- 在必要时拆成多个子查询

典型例子：

```text
原问题：那个报名时间呢？
改写后：2026 年秋季英语四六级考试报名时间是什么时候？
```

### 4.4 输出审核 Agent

职责：

- 判断主回答是否脱离上下文
- 判断是否存在敏感输出、不当措辞、过度肯定
- 必要时将回答改写为更安全、更保守的版本

### 4.5 知识入库审核 Agent（P1）

职责：

- 审查用户上传知识是否存在敏感内容、垃圾文本、低质量内容
- 决定允许入库、拒绝入库或标记待人工确认

该模块非常适合作为后续简历亮点，但建议在问答审核链路稳定后再接入。

---

## 5. 检索编排设计

### 5.1 Dense Retrieval

即当前项目已有的向量检索能力：

- 输入：改写后的查询
- 输出：基于语义相似度的候选文档

优点：

- 对语义表达、近义表述更友好

缺点：

- 对专有词、精确关键词、数字日期等有时不够稳

### 5.2 Sparse Retrieval

建议优先使用 PostgreSQL Full Text Search 实现，而不是额外引入新的搜索基础设施。

优点：

- 利用当前已有 PostgreSQL 技术栈，改造成本低
- 对关键词、专有名词、时间、规则文本更友好

### 5.3 Multi-query Retrieval

由查询理解 Agent 生成多个检索查询，再并行检索：

```text
用户问题：四六级报名是什么时候，怎么缴费？

生成子查询：
- 四六级报名时间
- 四六级报名缴费方式
- 四六级线上缴费要求
```

优点：

- 提高召回率
- 解决单问题包含多意图的情况

### 5.4 HyDE Retrieval（P1）

由 LLM 先生成“假设答案”，再用假设答案做语义检索。

适用情况：

- 用户问题过短
- 用户表达过于模糊
- 原始问题和知识库表述风格差异较大

建议：

- 先保留为可扩展策略，不作为第一阶段默认路径

### 5.5 融合排序

建议采用 `RRF（Reciprocal Rank Fusion）` 融合多路检索结果。

融合来源可包括：

- Dense Retrieval
- Sparse Retrieval
- Multi-query Dense Results
- Multi-query Sparse Results

融合后再做：

- 文档去重
- 分数裁剪
- TopK 截断

### 5.6 上下文压缩

目的不是“摘要好看”，而是“减少噪音，提高最终回答命中率”。

可分三步：

- 关键句提取：优先保留与问题直接相关的句子
- 无关段落过滤：去掉背景描述、重复句、弱相关内容
- 结果融合：将多个零散片段组织成更连贯的上下文

---

## 6. 推荐的模块拆分

为了符合当前仓库的分层习惯，建议按以下顺序扩展：

### 6.1 Core 层

新增或扩展 Port：

- `QueryRewriter`
- `ReviewService`
- `SparseRetriever`
- `RetrievalOrchestrator`
- `Reranker`（如第一期不单独实现，也可先合并到编排器内部）

### 6.2 Infrastructure 层

新增实现：

- `llm/query_rewriter.go`
- `llm/reviewer.go`
- `repository/fts_search.go` 或 `vector/fts_search.go`
- `retrieval/rrf.go`
- `retrieval/context_compressor.go`

### 6.3 Engine 层

把当前 `AgentEngine` 升级为：

- `QueryPipeline` 或 `AnswerPipeline`

职责改成：

- 串联审核、改写、检索、压缩、回答、复审

### 6.4 API 层

建议新增或扩展：

- `POST /api/v1/chat`
  - 支持返回 `citations`
  - 支持返回 `review_result`
  - 支持返回 `retrieval_strategy`
- `PUT /api/v1/agents/:id`
  - 后续可扩展 Agent 策略配置

---

## 7. 推荐的数据结构扩展

### 7.1 Agent 策略配置

建议为 Agent 增加可配置策略字段，便于体现“每个 Agent 的工作模式不同”：

```text
AgentPolicy
  EnableInputReview bool
  EnableOutputReview bool
  EnableKnowledgeReview bool
  EnableQueryRewrite bool
  EnableMultiQuery bool
  EnableHybridRetrieval bool
  EnableContextCompression bool
  DenseTopK int
  SparseTopK int
  FinalTopK int
```

说明：

- 个人私有化模式可关闭审核
- 多人协作或管理者场景可开启审核
- 这部分很适合后续做前端配置面板

### 7.2 Chat 响应扩展

建议在不破坏现有兼容性的前提下，逐步扩展：

```json
{
  "answer": "……",
  "citations": [
    {
      "chunk_id": "agent_1_doc_x_chunk_2",
      "content": "……",
      "score": 0.92,
      "source": "dense"
    }
  ],
  "review_result": {
    "input_passed": true,
    "output_passed": true,
    "risk_level": "low"
  },
  "retrieval_strategy": {
    "rewritten_query": "……",
    "used_multi_query": true,
    "used_hybrid_retrieval": true
  }
}
```

---

## 8. 推荐的开发阶段

### 8.1 P0：最小可展示版

目标：

- 做出“多 Agent 味道”最强、成本最低的一版

范围：

- 输入审核 Agent
- 查询改写 Agent
- Dense + Sparse 双路检索
- RRF 融合
- 回答结果附带引用

这一阶段做完后，项目已经可以明显区别于普通 RAG Demo。

### 8.2 P1：增强版

范围：

- 多查询扩展
- 输出审核 Agent
- 上下文压缩
- Agent 策略可配置

### 8.3 P2：亮点版

范围：

- 知识入库审核 Agent
- HyDE 检索
- 检索评测与对比报告
- 前端策略开关与审核展示

---

## 9. 推荐的半个月学习与落地路线

本路线以“边学边做、优先完成一条完整主线”为原则，不建议在半个月里同时追求过多高级特性。

### 第 1-2 天：读懂当前项目

目标：

- 完全理清当前 RAG 主链路
- 找到未来插入审核与编排的最佳位置

建议阅读：

- `internal/core/ports.go`
- `internal/engine/executor.go`
- `internal/core/knowledge/chunk_service.go`
- `internal/infrastructure/vector/pgvector.go`
- `internal/api/v1/handler.go`

你要回答的问题：

- 当前 query 是在哪里做 embedding 的
- 当前检索结果是在哪里组装的
- 审核链路插在哪一层最合适
- 查询改写放在 Handler、Engine 还是独立服务更好

### 第 3-4 天：补 RAG 编排基础知识

目标：

- 理解检索前、检索中、检索后的典型优化点

重点学习：

- Query Rewrite
- Hybrid Retrieval
- Multi-query
- RRF
- Context Compression

建议做法：

- 不追求全量吃透论文
- 先理解作用、放置位置、适用场景和代价

### 第 5-6 天：补审核 Agent 基础知识

目标：

- 理解输入审核、输出审核、知识审核的职责边界

重点学习：

- Guardrails
- Moderation
- Answer Verification
- 风险等级与降级策略

建议产出：

- 先写一个审核结果结构体草案
- 明确哪些内容是“拒绝”，哪些是“提示后继续”

### 第 7-8 天：先实现查询改写与检索双路召回

目标：

- 完成第一版能跑通的“查询理解 + 检索编排”

建议顺序：

1. Query Rewrite
2. PostgreSQL FTS
3. Dense + Sparse 双路召回
4. RRF 融合

### 第 9-10 天：实现输入审核 Agent

目标：

- 在主链路前增加可开关审核层

建议能力：

- 判断是否敏感
- 判断是否不当
- 判断是否注入攻击或越权请求

### 第 11-12 天：扩展响应与补测试

目标：

- 把“可解释结果”真正展示出来

建议补齐：

- `citations`
- `review_result`
- `retrieval_strategy`
- 单测：融合排序、审核开关、查询改写、FTS 检索

### 第 13-15 天：补文档、补演示、补评测

目标：

- 让项目从“能跑”变成“能讲”

建议输出：

- 架构图
- 一组对比样例
- README 中新增 Agent 化说明
- 简历项目描述草稿

---

## 10. 推荐学习资料

以下资料以“够用、贴近当前项目、可快速转化”为原则筛选。

### 10.1 RAG 总体架构

- Microsoft Learn: Build advanced retrieval-augmented generation systems

学习重点：

- 查询预处理
- 检索增强
- 后处理与评估

### 10.2 向量检索

- `pgvector` 官方 README

学习重点：

- 向量距离
- 索引方式
- 检索语法

### 10.3 关键词检索

- PostgreSQL Full Text Search 官方文档

学习重点：

- `tsvector`
- `tsquery`
- 排序
- 词典与同义词

### 10.4 Query Rewrite / HyDE / 多查询

- Microsoft 高级 RAG 文档中对应章节
- HyDE 论文摘要即可

学习重点：

- 不同策略分别解决什么问题
- 哪些适合第一期直接落地

### 10.5 审核与 Guardrails

- OpenAI Cookbook 中的 Guardrails / Moderation / Evals 示例

学习重点：

- 如何设计可解释审核结果
- 如何做拒绝、降级与告警

### 10.6 融合排序

- RRF 相关资料

学习重点：

- 为什么多路检索结果不能直接拼接
- 为什么排序融合比简单拼接更稳

---

## 11. 我认为最可行的首期交付组合

如果只用半个月，最建议的首期交付组合是：

### 方案 A：最平衡

- 输入审核 Agent
- 查询改写 Agent
- Dense + Sparse 检索
- RRF 融合
- 引用结果返回

优点：

- Agent 味道明显
- 检索增强也有体现
- 改造范围可控

### 方案 B：更偏 Agent

- 输入审核 Agent
- 输出审核 Agent
- 查询改写 Agent
- 结果引用

优点：

- 多 Agent 协作更直观

缺点：

- 检索增强亮点不如方案 A 完整

综合推荐：优先做 `方案 A`，然后再追加输出审核 Agent。

---

## 12. 不建议第一期就做的事

以下内容不是没价值，而是投入产出比不适合当前阶段：

- 一上来就做 HyDE 默认主路径
- 一上来就做复杂权限系统
- 一上来就做分布式限流与高可用
- 一上来就引入额外搜索引擎替代 PostgreSQL
- 一上来就追求所有策略都可视化配置

原因：

- 当前目标是“做出有技术层次的 Agent 项目”
- 不是“做一个重型上线平台”

---

## 13. 最终结论

对于本项目，最值得推进的方向不是继续补普通业务功能，而是：

- 用多 Agent 审核链路体现 Agent 技术力
- 用检索编排体现 RAG 深度
- 用策略开关体现系统设计能力

在时间有限的情况下，首期建议聚焦：

- 输入审核 Agent
- 查询改写
- Hybrid Retrieval
- RRF
- 引用与策略元信息返回

这条路径最容易在半个月内形成“可实现、可测试、可展示、可写简历”的完整成果。
