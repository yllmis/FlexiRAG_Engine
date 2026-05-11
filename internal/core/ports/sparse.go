package ports

import (
	"context"
	"flexirag-engine/internal/core"
)

// SparseRetriever 定义稀疏检索（全文检索）的标准接口
// 与 VectorStore.Search 签名对齐，区别是输入为自然语言文本而非向量
type SparseRetriever interface {
	Search(ctx context.Context, agentID uint, query string, topK int) ([]core.SearchResult, error)
}
