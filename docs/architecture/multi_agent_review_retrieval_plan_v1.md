# 模块化审核与检索编排增强方案（v2）

本文档用于规划 `FlexiRAG Engine` 的下一阶段演进方向。基于当前讨论结果，本项目不再追求“全链路重智能编排”，而是采用：

- 模块化增强为主
- 必要时再做智能增强
- 规则配置优先
- 成本、时延、可维护性可控

该方案更适合当前项目定位：

- 偏个人学习与简历展示
- 没有固定真实业务场景
- 需要保留可扩展空间
- 不希望因过度智能编排导致 token 成本和复杂度失控

---

## 1. 方案目标

本期目标不是做一个重型智能编排平台，而是把当前单路向量检索问答系统升级成一个：

- 可配置的 RAG 能力增强系统
- 具备查询理解、混合检索、融合排序、上下文压缩能力
- 具备输入、输出、知识入库三类审核模块
- 能根据不同场景开关能力
- 在确有必要时再为个别模块增加智能判断能力

最终希望达到的展示效果：

- 不是“为了复杂而复杂”
- 而是“先做合理模块化设计，再对高价值环节进行智能增强”

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

- 只有单路向量检索，缺少检索前后处理
- 没有关键词检索或混合检索
- 没有查询重写和检索策略控制
- 没有审核模块
- 没有引用、校验、上下文压缩
- 没有“按场景配置能力”的机制

---

## 3. 核心设计原则

### 3.1 模块优先

优先建设稳定、低成本、可测试的模块：

- 查询理解模块
- 检索编排模块
- 融合排序模块
- 上下文压缩模块
- 审核模块

### 3.2 智能增强按需引入

只有满足以下条件时，才为某个模块增加额外的智能判断逻辑：

- 该模块需要进行明显的策略决策
- 该模块需要多步推理
- 该模块需要动态选择后续流程
- 规则和模板难以覆盖

例如：

- 简单查询重写：更适合叫“查询理解模块”
- 查询分类 + 多查询扩展 + 检索路由：可以升级为“检索策略路由模块”

### 3.3 规则优先于模型

在审核能力上，优先采用：

```text
业务规则/敏感词/正则
  -> 命中则直接处理
  -> 灰区再进入模型判断
```

原因：

- 成本低
- 响应快
- 可解释
- 适合没有真实场景的项目

### 3.4 场景开关化

不同使用模式下，对能力的需求不同：

- 个人私有化：默认关闭重审核，强调低成本与流畅体验
- 多人共享：开启输入审核、输出审核、入库审核等控制项

---

## 4. 推荐总体架构

推荐将系统升级为以下模块化管线：

```text
用户请求
  -> 输入审核模块（可选）
  -> 查询理解模块
       - 查询清洗
       - 查询改写
       - 查询分类
       - 可选：多查询扩展
  -> 检索编排模块
       - Dense Retrieval
       - Sparse Retrieval(PostgreSQL FTS)
       - 可选：HyDE / Multi-query
  -> 融合排序模块
       - RRF
       - 去重
       - TopK 裁剪
  -> 上下文压缩模块
       - 关键句提取
       - 噪音过滤
       - 上下文融合
  -> 主回答模块
  -> 输出校验模块（可选）
  -> 返回答案 + 引用 + 审核结果 + 检索策略元数据
```

该架构的重点不是“堆很多智能环节”，而是：

- 流程合理
- 职责清晰
- 组件可替换
- 成本可控

---

## 5. 各模块职责定义

### 5.1 查询理解模块

职责：

- 清洗用户输入
- 在必要时做查询改写
- 识别问题类型
- 生成适合检索的查询表达

说明：

- 如果只是简单改写，不建议强行包装成智能角色
- 更准确的名称是 `Query Understanding Module`

可升级条件：

- 需要根据问题类型决定走 Dense / Sparse / Hybrid
- 需要生成多查询
- 需要决定是否进入审核、是否拒答

满足这些条件后，可以升级成“带策略决策的检索路由模块”。

### 5.2 检索编排模块

职责：

- 调用不同检索方式获取候选文档
- 目前优先支持：
  - 向量检索
  - PostgreSQL FTS

说明：

- 这是当前项目效果提升最直接的模块
- 相比复杂智能编排，这部分更值得优先做

### 5.3 融合排序模块

职责：

- 融合多路检索结果
- 去重
- 排序
- 截断

建议第一期采用：

- `RRF（Reciprocal Rank Fusion）`

### 5.4 上下文压缩模块

职责：

- 从检索结果中提取更有价值的片段
- 去掉明显无关内容
- 控制上下文长度

作用：

- 减少噪音
- 降低“迷失在中间”问题
- 提升回答集中度

### 5.5 主回答模块

职责：

- 基于最终上下文生成答案
- 遵守 Agent 的 `system_prompt`
- 输出引用或依据

说明：

- 当前 `AgentEngine` 可以继续承担这一职责
- 后续只需把输入从“单一路检索结果”升级成“编排后上下文”

### 5.6 输入审核模块

职责：

- 在用户提问前执行规则检查
- 重点关注：
  - prompt injection 特征
  - 明显违规词
  - 业务敏感词
  - 超长垃圾输入

说明：

- 第一阶段应优先做成规则模块
- 不建议默认做成独立智能审核链路

### 5.7 输出校验模块

职责：

- 在模型回答后执行规则检查
- 判断是否存在：
  - 敏感输出
  - 越界内容
  - 不希望泄露的业务词
  - 明显不符合约束的回答

说明：

- 第一阶段可先做规则校验
- 后续若需要可引入轻量模型兜底

### 5.8 入库审核模块

职责：

- 在知识入库前审查文本内容
- 重点关注：
  - 敏感内容
  - 低质量文本
  - 明显不应入库的信息

说明：

- 这个模块很适合作为“规则配置能力”的展示点
- 在没有真实业务场景时，默认走人工配置规则更合理

---

## 6. 审核能力设计：模块而非默认智能审核链路

### 6.1 为什么当前不优先做复杂审核链路

原因：

- 简单自用或公司内部 RAG，不需要每次都多调用一次模型审核
- 多次串行审核会显著增加 token 消耗和时延
- 没有真实场景时，审核策略本身更适合规则驱动

因此当前建议是：

- 先做 `审核模块`
- 后做 `审核智能兜底`
- 最后按需增加额外智能判断

### 6.2 推荐的分层审核方式

```text
第一层：规则审核
  - 敏感词
  - 正则
  - prompt injection 特征词
  - 长度和垃圾内容规则

第二层：灰区判断（后续可选）
  - 仅对规则无法确定的内容调用模型

第三层：动作执行
  - allow
  - warn
  - review
  - reject
```

### 6.3 适合当前项目的处理动作

建议只保留四种动作，降低复杂度：

- `allow`
- `warn`
- `review`
- `reject`

这样既够用，也便于写日志、做前端展示和补测试。

---

## 7. 审核规则配置方案

当前没有固定真实业务场景，因此审核规则应设计为“人工可维护、按目标自行配置”。

推荐采用两份文件：

### 7.1 文档文件

路径建议：

- `docs/rules/review_rules.md`

作用：

- 说明规则目的
- 说明适用场景
- 方便人工阅读和维护

### 7.2 机器规则文件

路径建议：

- `configs/review_rules.yaml`

作用：

- 供程序启动时读取
- 供审核模块执行匹配

### 7.3 为什么不用单独只靠 md

`md` 适合说明，不适合长期作为机器执行格式，因为：

- 不够结构化
- 不利于做字段校验
- 不便于支持作用域、动作、优先级、正则等扩展

因此建议：

```text
md 负责说明
yaml 负责执行
```

### 7.4 建议的规则结构

建议最低配字段如下：

```yaml
enabled: true

scopes:
  input: true
  output: true
  ingest: true

rules:
  - id: prompt_injection_001
    name: 忽略系统提示词攻击
    pattern: "忽略以上规则"
    match_type: contains
    category: prompt_injection
    action: reject
    severity: high
    scopes: [input]

  - id: biz_sensitive_001
    name: 内部客户名单
    pattern: "客户名单"
    match_type: contains
    category: business_sensitive
    action: review
    severity: medium
    scopes: [ingest, output]
```

### 7.5 推荐支持的匹配方式

第一期建议只支持三种：

- `exact`
- `contains`
- `regex`

这样足够覆盖多数演示与基础业务场景。

### 7.6 建议的审核结果结构

建议统一输出：

```json
{
  "passed": false,
  "action": "reject",
  "severity": "high",
  "matched_rules": [
    {
      "id": "prompt_injection_001",
      "name": "忽略系统提示词攻击"
    }
  ],
  "message": "命中高风险规则，已拦截"
}
```

该结构适合：

- Handler 返回
- 日志落库
- 前端展示
- 后续接入模型兜底

---

## 8. 检索增强设计

### 8.1 Dense Retrieval

即当前已有向量检索能力：

- 输入：查询理解模块输出的 query
- 输出：语义相似的候选片段

优点：

- 适合近义表达
- 适合自然语言描述

### 8.2 Sparse Retrieval

建议优先使用 PostgreSQL Full Text Search：

- 不引入额外基础设施
- 与当前技术栈兼容
- 对关键词、专有名词、时间和规则文本更友好

### 8.3 Hybrid Retrieval

第一期最推荐实现的检索增强方式：

```text
query
  -> dense retrieval
  -> sparse retrieval
  -> rrf fusion
```

原因：

- 效果提升明显
- token 成本几乎不增加
- 很适合当前项目

#### 8.3.1 当前状态

当前项目已经具备：

- Dense Retrieval（pgvector 语义检索）
- Sparse Retrieval（PostgreSQL FTS 关键词检索）
- RRF Lite 融合骨架

当前实现位置：

- `internal/engine/executor.go`
- `internal/engine/rrf.go`

当前实现方式更准确地说是：

```text
多个子查询
  -> 所有 dense 结果先拼成一个 dense 总榜
  -> 所有 sparse 结果先拼成一个 sparse 总榜
  -> 对两条总榜做一次 RRF
```

这个版本已经解决了“dense 分数与 sparse 分数不可直接比较”的问题，但它还不是标准的全局 RRF。

当前主要局限：

- 丢失了“每个子查询都是一条独立 ranked list”的语义
- 多个子查询同时命中同一文档时，融合收益没有被完整保留
- 同一路内重复文档需要显式做首次命中去重
- 还没有保留检索来源信息（来自哪条 query、哪种 retrieval）

#### 8.3.2 目标状态：标准全局 RRF

标准目标形态如下：

```text
用户问题
  -> 查询理解模块
  -> 得到多个子查询 q1, q2, q3 ...
  -> q1-dense 形成一条 ranked list
  -> q1-sparse 形成一条 ranked list
  -> q2-dense 形成一条 ranked list
  -> q2-sparse 形成一条 ranked list
  -> ...
  -> 所有 ranked lists 一次性进入全局 RRF
  -> 得到最终 Top-K 候选
```

也就是说：

- RRF 的输入单位不是“dense 一路、sparse 一路”
- 而是“每个 子查询 × 检索方式 形成的一条独立排序列表”

对应的 RRF 公式保持不变：

```text
Score(d) = Σ 1 / (k + rank_i(d))
```

设计意图：

- 用排名而不是原始分数做融合，避免不同检索路径分数量纲不一致
- 保留多 query 命中信号
- 保留 dense / sparse 双路命中信号
- 在不引入额外模型调用的前提下提升融合排序质量

#### 8.3.3 第一版落地边界（推荐）

建议这一版只做“标准 RRF 轻量实现”，不做过度扩展。

这一版一定要做：

1. 每个 `query × retrieval_type` 单独形成 ranked list
2. 所有 lists 一次性传入 `rrfFuse`
3. 同一条 list 内部重复文档只取首次排名
4. 融合后统一排序并截断 topK
5. 补单测锁住排序语义

这一版先不要做：

- 动态权重
- dense / sparse 人工加权
- 学习排序
- 重型 rerank
- 动态调整 `rrfK`

原因：

- 先把 RRF 的核心语义做对
- 控制工程复杂度
- 保持项目“模块化增强优先”的路线

#### 8.3.4 后续增强方向

标准 RRF 做稳之后，再考虑下面几项：

1. `source/citations` 返回
   - 标记结果来自 dense、sparse，还是双路同时命中
   - 标记命中的子查询来源
2. 上下文压缩
   - 在最终 Top-K 后做去噪、摘要、重点句提取
3. 中文 sparse 检索增强
   - 当前 PostgreSQL `simple` FTS 更适合作为关键词补充检索
   - 若后续要强化中文能力，再考虑中文分词或 trigram 兜底

#### 8.3.5 验收标准

完成标准 RRF 轻量实现后，应满足：

- Dense 与 Sparse 原始分数不再直接混排比较
- 同一文档在多个子查询、多种检索路径中命中时，能够正确累计 RRF 分数
- 同一条 list 内重复文档不会重复累计票数
- RRF 单测能覆盖：
  - 多 query、多 list 融合
  - 单 list 重复 ID 去重
  - Top-K 截断
  - 同分稳定排序
  - 空输入与退化路径

### 8.4 Multi-query（P1）

说明：

- 适合用户问题包含多个意图时
- 但会增加检索次数

建议：

- 先作为增强项，不作为第一期默认能力

### 8.5 HyDE（P2）

说明：

- 技术亮点足够强
- 但会增加一次模型调用成本

建议：

- 第一阶段先理解，不急于落地

---

## 9. 推荐的开发优先级

### 9.1 第一优先级：检索增强

建议顺序：

1. 查询理解模块
2. Dense + Sparse Retrieval
3. RRF 融合
4. 上下文压缩
5. 引用结果返回

原因：

- 最直接提升回答质量
- 不会明显提高 token 成本
- 最符合“模块化增强”的路线

### 9.2 第二优先级：审核模块

建议顺序：

1. 输入审核模块
2. 输出校验模块
3. 入库审核模块

原因：

- 适合做成可配置能力
- 便于体现项目的可扩展性和可控性

### 9.3 第三优先级：智能增强升级

只在确实需要时再做：

- 检索策略路由模块
- 灰区审核模型兜底
- 生成后轻量校验逻辑

---

## 10. 推荐的模块拆分

为了符合当前仓库的分层习惯，建议按以下顺序扩展：

### 10.1 Core 层

新增或扩展 Port：

- `QueryUnderstandingService`
- `ReviewService`
- `SparseRetriever`
- `RetrievalOrchestrator`
- `ContextCompressor`

### 10.2 Infrastructure 层

新增实现：

- `llm/query_rewriter.go`
- `review/rule_loader.go`
- `review/rule_matcher.go`
- `retrieval/fts_search.go`
- `retrieval/rrf.go`
- `retrieval/context_compressor.go`

### 10.3 Engine 层

建议把当前 `AgentEngine` 升级为：

- `AnswerPipeline`

职责改成：

- 串联查询理解、检索编排、上下文压缩、回答、输出校验

### 10.4 API 层

建议扩展：

- `POST /api/v1/chat`
  - 返回 `answer`
  - 返回 `citations`
  - 返回 `review_result`
  - 返回 `retrieval_strategy`
- `POST /api/v1/knowledge/ingest`
  - 可扩展返回入库审核结果

---

## 11. 推荐的数据结构扩展

### 11.1 能力策略配置

建议为 Agent 业务实体增加策略字段：

```text
AgentPolicy
  EnableInputReview bool
  EnableOutputReview bool
  EnableKnowledgeReview bool
  EnableQueryRewrite bool
  EnableHybridRetrieval bool
  EnableContextCompression bool
  DenseTopK int
  SparseTopK int
  FinalTopK int
```

说明：

- 这比在链路里人为增加许多智能角色更实用
- 更符合“按场景开关能力”的思路

### 11.2 Chat 响应扩展

建议逐步扩展为：

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
    "action": "allow"
  },
  "retrieval_strategy": {
    "rewritten_query": "……",
    "used_hybrid_retrieval": true
  }
}
```

---

## 12. 推荐的半个月学习与落地路线

### 第 1-2 天：吃透当前项目

重点看：

- `internal/core/ports.go`
- `internal/engine/executor.go`
- `internal/core/knowledge/chunk_service.go`
- `internal/infrastructure/vector/pgvector.go`
- `internal/api/v1/handler.go`

目标：

- 理清当前链路
- 确认每个增强模块应该插在哪一层

### 第 3-4 天：补 RAG 编排基础

重点学习：

- Query Rewrite
- Hybrid Retrieval
- RRF
- Context Compression

目标：

- 能画出目标链路
- 能解释每个模块解决什么问题

### 第 5-6 天：补审核模块基础

重点学习：

- 规则审核
- 敏感词配置
- Prompt Injection 特征
- 审核动作设计

目标：

- 能设计 `review_rules.yaml`
- 能设计统一审核结果结构

### 第 7-10 天：先做第一条主线

推荐顺序：

1. 查询理解模块
2. PostgreSQL FTS
3. Dense + Sparse 检索
4. RRF 融合
5. 引用返回

### 第 11-12 天：补审核模块

推荐顺序：

1. 输入审核
2. 输出校验
3. 入库审核骨架

### 第 13-15 天：补测试与文档

输出目标：

- 测试
- 架构图
- 样例对比
- README 补充
- 简历描述

---

## 13. 首期最推荐交付组合

如果只用半个月，最推荐做下面这条线：

- 查询理解模块
- Dense + Sparse Hybrid Retrieval
- RRF 融合
- 上下文压缩
- 输入审核模块
- 引用结果返回

这是当前阶段最平衡的方案。

优点：

- 能明显提升效果
- 不会引入过重智能编排开销
- 能体现架构判断力
- 更适合简历与面试讲述

---

## 14. 当前不建议优先做的事

以下内容不是没有价值，而是当前投入产出比不高：

- 一开始就做很多独立智能环节串行协作
- 一开始就默认开启 HyDE
- 一开始就做复杂审核模型链
- 一开始就做外部搜索系统接入
- 一开始就做高可用和分布式能力

原因：

- 当前目标是做出“合理、清晰、可展示”的增强版 RAG 系统
- 不是做一个重型生产平台

---

## 15. 最终结论

基于当前项目定位和讨论结果，下一阶段最合理的方向不是“重智能编排”，而是：

- 先用模块化增强提升核心能力
- 先做检索增强，再做审核模块
- 审核优先采用人工可配置规则
- 用 `md + yaml` 的方式管理审核说明和机器规则
- 仅在确实需要决策和路由的环节再增加智能判断

对当前项目来说，最值得优先推进的是：

- 查询理解模块
- Hybrid Retrieval
- RRF 融合
- 上下文压缩
- 输入/输出/入库审核模块

这条路线既能体现技术深度，又不会因为过度智能编排导致 token 消耗、时延和维护复杂度显著上升。
