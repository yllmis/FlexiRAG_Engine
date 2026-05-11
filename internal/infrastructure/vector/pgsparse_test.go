package vector

import (
	"context"
	"os"
	"testing"

	"flexirag-engine/internal/infrastructure/database"

	"gorm.io/gorm"
)

// setupPGDB 连接测试用 PostgreSQL，环境变量 TEST_PG_DSN 为空则跳过
func setupPGDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("TEST_PG_DSN") == "" {
		t.Skip("TEST_PG_DSN 未设置，跳过 PostgreSQL 集成测试")
	}
	db, err := database.NewPostgresDB(database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "flexirag_test",
		SSLMode:  "disable",
		TimeZone: "Asia/Shanghai",
	})
	if err != nil {
		t.Skipf("连接 PostgreSQL 失败，跳过: %v", err)
	}
	return db
}

func TestPGSparseRetriever_Search(t *testing.T) {
	db := setupPGDB(t)

	// 初始化 VectorStore（创建表 + trigger + GIN 索引）
	store, err := NewPGVectorStore(db)
	if err != nil {
		t.Fatalf("初始化 PGVectorStore 失败: %v", err)
	}

	sparse := NewPGSparseRetriever(db)
	ctx := context.Background()

	// 清理测试数据
	t.Cleanup(func() {
		db.Exec("DELETE FROM document_chunks WHERE id LIKE 'fts_test_%'")
	})

	// 插入测试数据（通过 Upsert，trigger 自动填充 content_tsv）
	testData := []struct {
		id      string
		content string
	}{
		{"fts_test_1", "FlexiRAG 是一个企业级 RAG 引擎，支持多租户隔离"},
		{"fts_test_2", "PostgreSQL 提供了强大的全文检索能力"},
		{"fts_test_3", "向量检索擅长语义匹配，全文检索擅长关键词匹配"},
		{"fts_test_4", "pgvector 扩展为 PostgreSQL 添加了向量相似度搜索"},
		{"fts_test_5", "这是一个不相关的文档，用于验证检索不会误命中"},
	}

	for _, d := range testData {
		err := store.Upsert(ctx, d.id, []float32{0.1, 0.2}, map[string]interface{}{
			"agent_id": uint(1),
			"content":  d.content,
		})
		if err != nil {
			t.Fatalf("插入测试数据失败 [%s]: %v", d.id, err)
		}
	}

	// 测试 1：关键词检索 "全文检索"
	t.Run("关键词命中", func(t *testing.T) {
		results, err := sparse.Search(ctx, 1, "全文检索", 10)
		if err != nil {
			t.Fatalf("FTS 检索失败: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("期望命中包含'全文检索'的文档，结果为空")
		}
		// 验证结果中包含预期文档
		found := false
		for _, r := range results {
			if r.ID == "fts_test_2" {
				found = true
				if r.Score <= 0 {
					t.Fatalf("期望 score > 0，实际: %f", r.Score)
				}
			}
		}
		if !found {
			t.Fatal("未命中 fts_test_2（包含'全文检索'）")
		}
	})

	// 测试 2：多词检索 "PostgreSQL 向量"
	t.Run("多词检索", func(t *testing.T) {
		results, err := sparse.Search(ctx, 1, "PostgreSQL 向量", 10)
		if err != nil {
			t.Fatalf("FTS 检索失败: %v", err)
		}
		// 应该命中 fts_test_4（同时包含 PostgreSQL 和向量）
		found := false
		for _, r := range results {
			if r.ID == "fts_test_4" {
				found = true
			}
		}
		if !found {
			t.Fatal("未命中 fts_test_4（包含'PostgreSQL'和'向量'）")
		}
	})

	// 测试 3：不存在的关键词
	t.Run("无命中", func(t *testing.T) {
		results, err := sparse.Search(ctx, 1, "区块链量子计算", 10)
		if err != nil {
			t.Fatalf("FTS 检索失败: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("期望无命中，实际命中 %d 条", len(results))
		}
	})

	// 测试 4：topK 限制
	t.Run("topK限制", func(t *testing.T) {
		results, err := sparse.Search(ctx, 1, "检索", 2)
		if err != nil {
			t.Fatalf("FTS 检索失败: %v", err)
		}
		if len(results) > 2 {
			t.Fatalf("期望最多 2 条结果，实际 %d 条", len(results))
		}
	})

	// 测试 5：agent_id 隔离
	t.Run("租户隔离", func(t *testing.T) {
		results, err := sparse.Search(ctx, 999, "全文检索", 10)
		if err != nil {
			t.Fatalf("FTS 检索失败: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("agent_id=999 不应命中任何结果，实际 %d 条", len(results))
		}
	})
}

func TestPGSparseRetriever_TopKZero(t *testing.T) {
	db := setupPGDB(t)
	sparse := NewPGSparseRetriever(db)
	ctx := context.Background()

	results, err := sparse.Search(ctx, 1, "test", 0)
	if err != nil {
		t.Fatalf("topK=0 不应报错: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("topK=0 应返回空结果，实际 %d 条", len(results))
	}
}
