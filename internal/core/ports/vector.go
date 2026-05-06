package ports

import (
	"context"
	"flexirag-engine/internal/core"
)

// VectorStore 定义了向量数据库的操作标准
// 无论底层是 Milvus, Pinecone 还是 pgvector，都要实现这个接口
type VectorStore interface {
	// Upsert: 插入或更新向量
	Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error

	// Search: 根据向量搜索相似内容
	Search(ctx context.Context, agentId uint, vector []float32, topK int) ([]core.SearchResult, error)

	// Delete: 根据 ID 删除向量
	Delete(ctx context.Context, id string) error
}