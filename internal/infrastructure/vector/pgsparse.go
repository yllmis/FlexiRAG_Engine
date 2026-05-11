package vector

import (
	"context"
	"fmt"

	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/ports"

	"gorm.io/gorm"
)

var _ ports.SparseRetriever = (*PGSparseRetriever)(nil)

// PGSparseRetriever 基于 PostgreSQL Full-Text Search 的稀疏检索实现
type PGSparseRetriever struct {
	db *gorm.DB
}

func NewPGSparseRetriever(db *gorm.DB) *PGSparseRetriever {
	return &PGSparseRetriever{db: db}
}

// Search 执行全文检索，返回按 ts_rank 降序排列的结果
//
//	底层 SQL:
//	SELECT id, content, ts_rank(content_tsv, q) AS score
//	FROM document_chunks, plainto_tsquery('simple', $query) AS q
//	WHERE agent_id = $agentID AND content_tsv @@ q
//	ORDER BY score DESC
//	LIMIT $topK
func (s *PGSparseRetriever) Search(ctx context.Context, agentID uint, query string, topK int) ([]core.SearchResult, error) {
	if topK <= 0 {
		return []core.SearchResult{}, nil
	}

	var records []struct {
		ID      string
		Content string
		Score   float32
	}

	// plainto_tsquery 自动将自然语言转为 tsquery（处理 AND/OR/NOT 逻辑）
	// 'simple' 配置不分词，按字面匹配，适合中文场景
	sql := `
		SELECT id, content, ts_rank(content_tsv, q) AS score
		FROM document_chunks, plainto_tsquery('simple', ?) AS q
		WHERE agent_id = ? AND content_tsv @@ q
		ORDER BY score DESC
		LIMIT ?
	`
	err := s.db.WithContext(ctx).
		Raw(sql, query, agentID, topK).
		Scan(&records).Error
	if err != nil {
		return nil, fmt.Errorf("FTS 检索失败: %w", err)
	}

	results := make([]core.SearchResult, len(records))
	for i, r := range records {
		results[i] = core.SearchResult{
			ID:      r.ID,
			Content: r.Content,
			Score:   r.Score,
		}
	}
	return results, nil
}
