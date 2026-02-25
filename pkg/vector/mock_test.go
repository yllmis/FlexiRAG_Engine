package vector

import (
	"context"
	"encoding/json"
	"testing"
)

func TestUpsertSameIDDifferentAgentNoOverwrite(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	if err := store.Upsert(ctx, "doc-1", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "A"}); err != nil {
		t.Fatalf("upsert agent1 failed: %v", err)
	}
	if err := store.Upsert(ctx, "doc-1", []float32{0, 1}, map[string]interface{}{"agent_id": uint(2), "content": "B"}); err != nil {
		t.Fatalf("upsert agent2 failed: %v", err)
	}

	res1, err := store.Search(ctx, 1, []float32{1, 0}, 1)
	if err != nil {
		t.Fatalf("search agent1 failed: %v", err)
	}
	res2, err := store.Search(ctx, 2, []float32{0, 1}, 1)
	if err != nil {
		t.Fatalf("search agent2 failed: %v", err)
	}

	if len(res1) != 1 || res1[0].Content != "A" {
		t.Fatalf("agent1 result mismatch: %+v", res1)
	}
	if len(res2) != 1 || res2[0].Content != "B" {
		t.Fatalf("agent2 result mismatch: %+v", res2)
	}
}

func TestSearchTopKLessOrEqualZero(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()
	_ = store.Upsert(ctx, "doc-1", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "A"})

	res, err := store.Search(ctx, 1, []float32{1, 0}, 0)
	if err != nil {
		t.Fatalf("topK=0 should not error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected empty result when topK=0, got: %+v", res)
	}

	res, err = store.Search(ctx, 1, []float32{1, 0}, -1)
	if err != nil {
		t.Fatalf("topK<0 should not error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected empty result when topK<0, got: %+v", res)
	}
}

func TestDimensionValidation(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	if err := store.Upsert(ctx, "doc-1", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "A"}); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	if err := store.Upsert(ctx, "doc-2", []float32{1, 0, 0}, map[string]interface{}{"agent_id": uint(1), "content": "B"}); err == nil {
		t.Fatal("expected dimension mismatch error on upsert")
	}
	if _, err := store.Search(ctx, 1, []float32{1, 0, 0}, 1); err == nil {
		t.Fatal("expected dimension mismatch error on search")
	}
}

func TestUpsertVectorDeepCopy(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	vec := []float32{1, 0}
	if err := store.Upsert(ctx, "doc-1", vec, map[string]interface{}{"agent_id": uint(1), "content": "A"}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	vec[0], vec[1] = 0, 1

	res, err := store.Search(ctx, 1, []float32{1, 0}, 1)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("unexpected result length: %+v", res)
	}
	if res[0].Score < 0.99 {
		t.Fatalf("expected score close to 1 after deep copy, got: %f", res[0].Score)
	}
}

func TestParseAgentIDVariants(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	cases := []map[string]interface{}{
		{"agent_id": uint(1), "content": "u"},
		{"agent_id": int(2), "content": "i"},
		{"agent_id": int64(3), "content": "i64"},
		{"agent_id": float64(4), "content": "f64"},
		{"agent_id": json.Number("5"), "content": "jn"},
		{"agent_id": "6", "content": "s"},
	}

	for idx, md := range cases {
		id := "doc-" + string(rune('a'+idx))
		if err := store.Upsert(ctx, id, []float32{1, 0}, md); err != nil {
			t.Fatalf("case %d should pass but failed: %v", idx, err)
		}
	}

	if err := store.Upsert(ctx, "bad", []float32{1, 0}, map[string]interface{}{"agent_id": -1}); err == nil {
		t.Fatal("expected invalid negative agent_id error")
	}
}

func TestSearchHonorsContextCancel(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()
	if err := store.Upsert(ctx, "doc-1", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "A"}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Search(canceledCtx, 1, []float32{1, 0}, 1); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestSearchStableOrderByIDWhenScoreEqual(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	_ = store.Upsert(ctx, "b", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "B"})
	_ = store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "A"})

	res, err := store.Search(ctx, 1, []float32{1, 0}, 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("unexpected result length: %+v", res)
	}
	if res[0].ID != "a" || res[1].ID != "b" {
		t.Fatalf("expected stable order by ID when score equal, got: %+v", res)
	}
}

func TestUpsertPrecomputesNorm(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()
	if err := store.Upsert(ctx, "doc-1", []float32{3, 4}, map[string]interface{}{"agent_id": uint(1), "content": "A"}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	store.mu.RLock()
	bucket := store.agents[1]
	store.mu.RUnlock()
	if bucket == nil {
		t.Fatal("bucket should exist")
	}

	bucket.mu.RLock()
	item, ok := bucket.items["doc-1"]
	bucket.mu.RUnlock()
	if !ok {
		t.Fatal("item should exist")
	}
	if item.Norm < 4.99 || item.Norm > 5.01 {
		t.Fatalf("expected cached norm ~= 5, got: %f", item.Norm)
	}
}

func TestSearchReturnsExactTopK(t *testing.T) {
	store := NewMockVectorStore()
	ctx := context.Background()

	_ = store.Upsert(ctx, "d1", []float32{1, 0}, map[string]interface{}{"agent_id": uint(1), "content": "1"})
	_ = store.Upsert(ctx, "d2", []float32{0.9, 0.1}, map[string]interface{}{"agent_id": uint(1), "content": "2"})
	_ = store.Upsert(ctx, "d3", []float32{0.8, 0.2}, map[string]interface{}{"agent_id": uint(1), "content": "3"})
	_ = store.Upsert(ctx, "d4", []float32{0.1, 0.9}, map[string]interface{}{"agent_id": uint(1), "content": "4"})

	res, err := store.Search(ctx, 1, []float32{1, 0}, 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].ID != "d1" || res[1].ID != "d2" {
		t.Fatalf("topK mismatch, got: %+v", res)
	}
}
