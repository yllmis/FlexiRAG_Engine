package vector

import (
	"context"
	"flexirag-engine/internal/core"
	"math"
	"sort"
	"sync"
)

var _ core.VectorStore = (*MockVectorStore)(nil)

// 存储在内存中的实体结构
type vectorItem struct {
	ID      string
	AgentID uint
	Values  []float32
	Content string
}

type MockVectorStore struct {
	mu    sync.RWMutex
	items map[string]vectorItem
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		items: make(map[string]vectorItem),
	}
}

// Upesert 插入或更新一个向量项
func (m *MockVectorStore) Upsert(ctx context.Context, id string, values []float32, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	content, _ := metadata["content"].(string)
	agentIDFloat, _ := metadata["agent_id"].(float64) //json解析时数字会被解析为float64
	agentID := uint(agentIDFloat)

	// 手动传入uint
	if val, ok := metadata["agent_id"].(uint); ok {
		agentID = val
	}

	m.items[id] = vectorItem{
		ID:      id,
		AgentID: agentID,
		Values:  values,
		Content: content,
	}
	return nil

}

// Search 根据查询向量返回相似的向量项(相似度计算)
func (m *MockVectorStore) Search(ctx context.Context, agentID uint, queryVector []float32, topK int) ([]core.SearchResult, error) {
	m.mu.RLock()

	// 1. 过滤并计算相似度
	var results []core.SearchResult
	for _, item := range m.items {
		if item.AgentID != agentID {
			continue
		}
		score := cosineSimilarity(queryVector, item.Values)
		results = append(results, core.SearchResult{
			ID:      item.ID,
			Content: item.Content,
			Score:   score,
		})
	}
	m.mu.RUnlock()

	// 2. 根据相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 3. 返回前topK个结果
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

// 计算两个向量之间的余弦相似度
func cosineSimilarity(vecA, vecB []float32) float32 {
	if len(vecA) != len(vecB) || len(vecA) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(vecA); i++ {
		dotProduct += vecA[i] * vecB[i]
		normA += vecA[i] * vecA[i]
		normB += vecB[i] * vecB[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
