package vector

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"flexirag-engine/internal/core"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
)

var _ core.VectorStore = (*MockVectorStore)(nil)

// 存储在内存中的实体结构
type vectorItem struct {
	ID      string
	AgentID uint
	Values  []float32
	Norm    float32
	Content string
}

type AgentBucket struct {
	mu    sync.RWMutex
	items map[string]vectorItem
}

type MockVectorStore struct {
	mu     sync.RWMutex
	agents map[uint]*AgentBucket
	dim    int
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		agents: make(map[uint]*AgentBucket),
	}
}

// Upsert 插入或更新一个向量项
func (m *MockVectorStore) Upsert(ctx context.Context, id string, values []float32, metadata map[string]interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return errors.New("id 不能为空")
	}
	if len(values) == 0 {
		return errors.New("向量不能为空")
	}

	agentID, err := parseAgentID(metadata)
	if err != nil {
		return err
	}
	content, _ := metadata["content"].(string)

	m.mu.Lock()
	if m.dim == 0 {
		m.dim = len(values)
	} else if len(values) != m.dim {
		m.mu.Unlock()
		return fmt.Errorf("向量维度不一致: 期望 %d，实际 %d", m.dim, len(values))
	}

	bucket, ok := m.agents[agentID]
	if !ok {
		bucket = &AgentBucket{items: make(map[string]vectorItem)}
		m.agents[agentID] = bucket
	}
	m.mu.Unlock()

	copiedValues := append([]float32(nil), values...)
	norm := vectorNorm(copiedValues)

	bucket.mu.Lock()
	bucket.items[id] = vectorItem{
		ID:      id,
		AgentID: agentID,
		Values:  copiedValues,
		Norm:    norm,
		Content: content,
	}
	bucket.mu.Unlock()

	return nil
}

// Search 根据查询向量返回相似的向量项（最小堆筛选 topK）
func (m *MockVectorStore) Search(ctx context.Context, agentID uint, queryVector []float32, topK int) ([]core.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if topK <= 0 {
		return []core.SearchResult{}, nil
	}
	if len(queryVector) == 0 {
		return nil, errors.New("查询向量不能为空")
	}

	m.mu.RLock()
	dim := m.dim
	bucket, ok := m.agents[agentID]
	m.mu.RUnlock()
	if !ok {
		return []core.SearchResult{}, nil
	}
	if dim > 0 && len(queryVector) != dim {
		return nil, fmt.Errorf("查询向量维度不一致: 期望 %d，实际 %d", dim, len(queryVector))
	}

	queryNorm := vectorNorm(queryVector)

	bucket.mu.RLock()
	h := make(resultMinHeap, 0, topK)
	heap.Init(&h)

	for _, item := range bucket.items {
		if err := ctx.Err(); err != nil {
			bucket.mu.RUnlock()
			return nil, err
		}

		score := cosineSimilarityWithNorm(queryVector, queryNorm, item.Values, item.Norm)
		candidate := core.SearchResult{
			ID:      item.ID,
			Content: item.Content,
			Score:   score,
		}

		if h.Len() < topK {
			heap.Push(&h, candidate)
			continue
		}

		// 若候选优于堆顶（当前最差），则替换
		if better(candidate, h[0]) {
			h[0] = candidate
			heap.Fix(&h, 0)
		}
	}
	bucket.mu.RUnlock()

	results := make([]core.SearchResult, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(&h).(core.SearchResult)
	}

	// 最终稳定排序，确保同分时输出一致
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})

	return results, nil
}

func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return nil
	}

	m.mu.RLock()
	buckets := make([]*AgentBucket, 0, len(m.agents))
	for _, bucket := range m.agents {
		buckets = append(buckets, bucket)
	}
	m.mu.RUnlock()

	for _, bucket := range buckets {
		bucket.mu.Lock()
		delete(bucket.items, id)
		bucket.mu.Unlock()
	}
	return nil
}

func parseAgentID(metadata map[string]interface{}) (uint, error) {
	if metadata == nil {
		return 0, errors.New("metadata 不能为空")
	}
	raw, ok := metadata["agent_id"]
	if !ok {
		return 0, errors.New("metadata 缺少 agent_id")
	}

	switch value := raw.(type) {
	case uint:
		return value, nil
	case uint64:
		return uint(value), nil
	case int:
		if value < 0 {
			return 0, errors.New("agent_id 不能为负数")
		}
		return uint(value), nil
	case int64:
		if value < 0 {
			return 0, errors.New("agent_id 不能为负数")
		}
		return uint(value), nil
	case float64:
		if value < 0 {
			return 0, errors.New("agent_id 不能为负数")
		}
		return uint(value), nil
	case json.Number:
		n, err := value.Int64()
		if err != nil || n < 0 {
			return 0, errors.New("agent_id 非法")
		}
		return uint(n), nil
	case string:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return 0, errors.New("agent_id 非法")
		}
		return uint(n), nil
	default:
		return 0, fmt.Errorf("不支持的 agent_id 类型: %T", raw)
	}
}

func vectorNorm(vec []float32) float32 {
	if len(vec) == 0 {
		return 0
	}
	var sum float32
	for i := 0; i < len(vec); i++ {
		sum += vec[i] * vec[i]
	}
	return float32(math.Sqrt(float64(sum)))
}

func cosineSimilarityWithNorm(vecA []float32, normA float32, vecB []float32, normB float32) float32 {
	if len(vecA) != len(vecB) || len(vecA) == 0 {
		return 0
	}
	if normA == 0 || normB == 0 {
		return 0
	}

	var dotProduct float32
	for i := 0; i < len(vecA); i++ {
		dotProduct += vecA[i] * vecB[i]
	}
	return dotProduct / (normA * normB)
}

// 仅用于堆内比较：更差的结果在堆顶
type resultMinHeap []core.SearchResult

func (h resultMinHeap) Len() int { return len(h) }
func (h resultMinHeap) Less(i, j int) bool {
	if h[i].Score == h[j].Score {
		return h[i].ID > h[j].ID
	}
	return h[i].Score < h[j].Score
}
func (h resultMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *resultMinHeap) Push(x interface{}) {
	*h = append(*h, x.(core.SearchResult))
}
func (h *resultMinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func better(a, b core.SearchResult) bool {
	if a.Score == b.Score {
		return a.ID < b.ID
	}
	return a.Score > b.Score
}
