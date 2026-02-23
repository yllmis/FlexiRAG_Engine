# 第2步：内存向量库（Mock）详细设计稿

## 1. 目标与边界

### 1.1 目标
- 在不依赖外部向量数据库的前提下，快速打通 RAG 检索闭环。
- 为后续替换 Milvus/pgvector 提供统一接口适配层。
- 保证多租户隔离（按 `agentID` 检索域隔离）。

### 1.2 边界（MVP）
- 仅支持单机内存存储。
- 仅支持精确检索（Brute Force），不引入 ANN 索引。
- 不做持久化，不做跨进程共享。

---

## 2. 数据结构设计

### 2.1 核心实体 `Record`
```text
Record {
  ID        string
  AgentID   uint
  Vector    []float32
  Norm      float32
  Content   string
  Metadata  map[string]any
  UpdatedAt int64
}
```

说明：
- `Norm` 预存向量范数，减少检索时重复计算。
- `Content` 直接用于回填到 `SearchResult`。
- `Metadata` 用于扩展来源信息（文档ID、chunk序号等）。

### 2.2 Agent 维度容器 `AgentBucket`
```text
AgentBucket {
  items map[string]*Record   // key = record.ID
}
```

### 2.3 存储实现 `MemoryVectorStore`
```text
MemoryVectorStore {
  mu     sync.RWMutex
  agents map[uint]*AgentBucket
  dim    int
}
```

说明：
- `agents` 以 `agentID` 为一级分区，天然隔离租户数据。
- `dim` 在首条入库后固定，后续写入必须维度一致。

---

## 3. 检索算法设计

### 3.1 相似度公式（余弦相似度）
$$
sim(q,v)=\frac{q\cdot v}{\|q\|\|v\|}
$$

### 3.2 Search 执行流程
1. 按 `agentID` 取对应 `AgentBucket`。
2. 计算查询向量 `query` 的范数。
3. 遍历 bucket 内所有 `Record`，计算余弦分数。
4. 使用大小为 `topK` 的最小堆维护候选集合。
5. 最终结果按 `score desc` 输出；同分时按 `ID asc` 保证稳定性。

### 3.3 关键策略
- 检索范围严格限制在单个 `agentID` 下。
- 空向量、维度不一致、`topK <= 0` 直接返回可解释错误/空结果。
- 检索循环中定期检查 `ctx.Done()`，支持超时取消。

---

## 4. 接口语义与行为约束

### 4.1 Upsert
- 输入：`agentID, id, vector, metadata`。
- 行为：存在则更新，不存在则插入。
- 约束：
  - 首次写入设置全局 `dim`。
  - 后续写入若维度不一致，返回错误。
  - 深拷贝 `vector` 和 `metadata`，防止外部修改导致竞态。

### 4.2 Search
- 输入：`agentID, vector, topK`。
- 行为：返回 `[]SearchResult{ID, Content, Score}`。
- 约束：
  - `topK <= 0` 返回空切片。
  - `agentID` 不存在返回空切片。
  - 结果稳定可复现（排序规则固定）。

### 4.3 Delete
- 输入：`id`（MVP 先与当前接口保持一致）。
- 行为：删除成功返回 `nil`。
- 建议：未命中也返回 `nil`，保证幂等性。

> 后续建议将 `Delete` 升级为 `(agentID, id)`，进一步强化多租户边界约束。

---

## 5. 并发安全方案

### 5.1 基础锁策略
- `Upsert/Delete`：使用 `mu.Lock()`。
- `Search`：使用 `mu.RLock()` 获取 bucket 引用后尽快释放，避免长时间读锁。

### 5.2 内存与竞态防护
- 入库时深拷贝可变数据（`[]float32`、`map`）。
- 输出时仅返回值对象，不暴露内部指针。
- 任何循环中的外部调用点都尊重 `context` 取消。

### 5.3 后续扩展（高并发）
- 分片锁（如 64 shards）降低全局锁冲突。
- 或每个 `AgentBucket` 独立 `RWMutex`。

### 潜在隐患 
- 隐患：快照引用的幻觉与 Map 崩溃
    原案： `mu.RLock()` 先拿到当前 `bucket` 快照引用，立即释放锁后计算分数（减少锁占用）。

- 排雷分析：
    在 Go 中，map 的引用只是一个指针。如果你获取了 `agents[agentID]` 这个 `bucket` 的引用，然后释放了全局读锁 `mu.RUnlock()`。紧接着你开始写 `for _, record := range bucket.items` 循环计算分数。
    如果在你循环的同时，另一个 Goroutine 调用了 Upsert 往这个 `bucket.items` 里并发写入了新数据，Go 运行时（Runtime）会立刻抛出无法被 recover 捕获的致命错误：
    `fatal error: concurrent map iteration and map write`
    整个服务器进程会瞬间崩溃。

- 修正方案：细粒度分级锁 (Per-Bucket RWMutex)
    为了实现“缩小锁粒度”的宏大目标，我们需要调整锁的层级：

```Go
// 全局引擎层
type MemoryVectorStore struct {
    mu     sync.RWMutex             // 仅保护 agents map 本身的增删
    agents map[uint]*AgentBucket    
    dim    int                      
}

// 租户桶层（核心改动：每个 Bucket 自带一把锁！）
type AgentBucket struct {
    mu    sync.RWMutex             // 保护本租户内部的 items 并发读写
    items map[string]*Record
}
```
- 检索流程变为：
  - 全局读锁 store.mu.RLock() -> 获取 bucket 指针 -> 全局读锁释放 store.mu.RUnlock()。（此时其他租户创建新 bucket 不受影响）。

  - 局部读锁 `bucket.mu.RLock() -> 遍历 bucket.items` 计算分数并塞入最小堆 -> 局部读锁释放 `bucket.mu.RUnlock()`。（此时该租户不能写入，但其他租户的读写完全并行）。


---

## 6. 复杂度评估

设：
- `N` = 全库向量数
- `n_a` = 单个 `agentID` 下向量数
- `d` = 向量维度
- `K` = topK

复杂度：
- `Upsert`：`O(d)`（拷贝 + 范数计算）
- `Search`：`O(n_a * d + n_a * logK)`
- `Delete`：均摊 `O(1)`
- 空间：`O(N * d)`（float32）

结论：MVP 阶段在中小规模数据下可接受，且实现简单、稳定、可验证。

---

## 7. 验收标准（Definition of Done）

- 多租户隔离：不同 `agentID` 检索结果完全隔离。
- 正确性：同输入同数据集多次检索结果一致。
- 并发安全：`go test -race` 无数据竞争。
- 可用性：上下文超时可中断，错误可追踪。
- 可替换性：未来替换 Milvus/pgvector 时无需改动 `engine` 业务逻辑。

---

## 8. 与当前项目结构的映射建议

- 接口定义：`internal/core/ports.go`
- 内存实现：建议放在 `pkg/vector/memory_store.go`
- 单元测试：建议放在 `pkg/vector/memory_store_test.go`
- 引擎调用：`internal/engine/executor.go` 仅依赖 `core.VectorStore`

该映射可确保“先跑通、后替换”的演进路径清晰。
