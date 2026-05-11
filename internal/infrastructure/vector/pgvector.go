package vector

import (
	"context"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/ports"
	"fmt"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ ports.VectorStore = (*PGVectorStore)(nil)

type DocumentChunk struct {
	ID         string `gorm:"primaryKey;type:varchar(255)"`
	AgentID    uint   `gorm:"index;not null"` // 加普通索引，加速多租户过滤
	Content    string `gorm:"type:text;not null"`
	ContentTsv string `gorm:"type:tsvector;column:content_tsv"` // PostgreSQL FTS 向量
	// 这样设计是为了兼容不同厂商的模型（OpenAI 1536维，GLM 可能 1024维）
	Embedding pgvector.Vector `gorm:"type:vector"`
}

// TableName 覆盖 GORM 默认的表名策略
func (DocumentChunk) TableName() string {
	return "document_chunks"
}

type PGVectorStore struct {
	db *gorm.DB
}

func NewPGVectorStore(db *gorm.DB) (*PGVectorStore, error) {

	err := db.AutoMigrate(&DocumentChunk{})
	if err != nil {
		return nil, err
	}

	// 初始化 PostgreSQL FTS：trigger 自动维护 content_tsv + GIN 索引
	if err := initFTS(db); err != nil {
		return nil, err
	}

	return &PGVectorStore{db: db}, nil
}

// initFTS 创建 trigger 函数和 GIN 索引，确保 content_tsv 与 content 列自动同步
func initFTS(db *gorm.DB) error {
	// 1. 回填已有行的 content_tsv
	if err := db.Exec(`UPDATE document_chunks SET content_tsv = to_tsvector('simple', content) WHERE content_tsv IS NULL`).Error; err != nil {
		return fmt.Errorf("回填 content_tsv 失败: %w", err)
	}

	// 2. 创建 trigger 函数：INSERT/UPDATE 时自动从 content 计算 content_tsv
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION document_chunks_tsv_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsv := to_tsvector('simple', NEW.content);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		return fmt.Errorf("创建 FTS trigger 函数失败: %w", err)
	}

	// 3. 绑定 trigger（DROP IF EXISTS 保证幂等）
	if err := db.Exec(`
		DROP TRIGGER IF EXISTS tsvector_update ON document_chunks;
		CREATE TRIGGER tsvector_update
			BEFORE INSERT OR UPDATE OF content ON document_chunks
			FOR EACH ROW EXECUTE FUNCTION document_chunks_tsv_trigger();
	`).Error; err != nil {
		return fmt.Errorf("创建 FTS trigger 失败: %w", err)
	}

	// 4. 创建 GIN 索引（IF NOT EXISTS 保证幂等）
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_content_tsv ON document_chunks USING GIN (content_tsv)`).Error; err != nil {
		return fmt.Errorf("创建 FTS GIN 索引失败: %w", err)
	}

	return nil
}

func (p *PGVectorStore) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	agentIDFloat, ok := metadata["agent_id"].(float64)
	if !ok {
		// 尝试转 uint (应对直接传 uint 的情况)
		if val, isUint := metadata["agent_id"].(uint); isUint {
			agentIDFloat = float64(val)
		} else {
			return fmt.Errorf("metadata 中缺少或错误的 agent_id")
		}
	}
	content, _ := metadata["content"].(string)

	chunk := DocumentChunk{
		ID:        id,
		AgentID:   uint(agentIDFloat),
		Content:   content,
		Embedding: pgvector.NewVector(vector),
	}

	// 对应 SQL: INSERT ... ON CONFLICT (id) DO UPDATE SET ...
	// content_tsv 不在此处填充——由 PostgreSQL trigger 自动从 content 列计算
	return p.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"agent_id", "content", "embedding"}),
		}).
		Create(&chunk).Error

}

func (p *PGVectorStore) Search(ctx context.Context, agentID uint, queryVector []float32, topK int) ([]core.SearchResult, error) {
	var records []struct {
		ID      string
		Content string
		Score   float32
	}

	// pgvector 转化
	qVec := pgvector.NewVector(queryVector)

	// 【核心算法转换】
	// pgvector 的 `<=>` 符号计算的是“余弦距离 (Cosine Distance)”。距离越小，代表越相似。
	// 但我们的接口定义的是“余弦相似度 (Cosine Similarity)”，相似度越大越好。
	// 数学公式：Cosine Similarity = 1 - Cosine Distance
	err := p.db.WithContext(ctx).
		Model(&DocumentChunk{}).
		Select("id, content, 1 - (embedding <=> ?) AS score", qVec).
		Where("agent_id = ?", agentID).
		Order(gorm.Expr("embedding <=> ?", qVec)). // 按距离升序排列（最相似的在最前面）
		Limit(topK).
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("PG 向量检索失败：%w", err)
	}

	results := make([]core.SearchResult, len(records))
	for i, rec := range records {
		results[i] = core.SearchResult{
			ID:      rec.ID,
			Content: rec.Content,
			Score:   rec.Score,
		}
	}

	return results, nil
}

// Delete 删除向量
func (p *PGVectorStore) Delete(ctx context.Context, id string) error {
	return p.db.WithContext(ctx).Delete(&DocumentChunk{ID: id}).Error
}
