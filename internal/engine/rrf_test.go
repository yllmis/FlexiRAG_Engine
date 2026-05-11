package engine

import (
	"flexirag-engine/internal/core"
	"testing"
)

func TestRRFFuse_TwoListCrossMatch(t *testing.T) {
	// doc_A 在两路都排第一 → RRF 分数最高
	// doc_B 只在 dense 排第一
	// doc_C 只在 sparse 排第一
	dense := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
	}
	sparse := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.5},
		{ID: "C", Content: "C", Score: 0.4},
	}

	results := rrfFuse([][]core.SearchResult{dense, sparse}, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// A 在两路都排名 1，RRF 分数 = 1/61 + 1/61 = 2/61 ≈ 0.0328
	// B 只在 dense 排名 1，RRF 分数 = 1/61 ≈ 0.0164
	// C 只在 sparse 排名 1，RRF 分数 = 1/61 ≈ 0.0164
	if results[0].ID != "A" {
		t.Fatalf("expected A at rank 1, got %s (score=%f)", results[0].ID, results[0].Score)
	}
	// B 和 C 同分，按 ID 排序
	if results[1].ID != "B" || results[2].ID != "C" {
		t.Fatalf("expected B then C, got %s then %s", results[1].ID, results[2].ID)
	}
}

func TestRRFFuse_SingleList(t *testing.T) {
	list := []core.SearchResult{
		{ID: "X", Content: "X", Score: 0.9},
		{ID: "Y", Content: "Y", Score: 0.8},
		{ID: "Z", Content: "Z", Score: 0.7},
	}

	results := rrfFuse([][]core.SearchResult{list}, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// 单路时排名不变
	if results[0].ID != "X" || results[1].ID != "Y" || results[2].ID != "Z" {
		t.Fatalf("order mismatch: %v", results)
	}
	// 验证 RRF 分数正确
	expected := 1.0 / (float32(rrfK) + 1) // rank 1 → 1/61
	if results[0].Score != expected {
		t.Fatalf("expected score %f, got %f", expected, results[0].Score)
	}
}

func TestRRFFuse_TopKTruncate(t *testing.T) {
	list := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
		{ID: "C", Content: "C", Score: 0.7},
	}

	results := rrfFuse([][]core.SearchResult{list}, 2)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "A" || results[1].ID != "B" {
		t.Fatalf("topK truncation mismatch: %v", results)
	}
}

func TestRRFFuse_EmptyInput(t *testing.T) {
	results := rrfFuse([][]core.SearchResult{}, 10)
	if len(results) != 0 {
		t.Fatalf("expected empty, got %v", results)
	}

	results = rrfFuse([][]core.SearchResult{{}}, 10)
	if len(results) != 0 {
		t.Fatalf("expected empty, got %d", len(results))
	}
}

func TestRRFFuse_TopKZero(t *testing.T) {
	list := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
	}
	results := rrfFuse([][]core.SearchResult{list}, 0)
	if len(results) != 0 {
		t.Fatalf("expected empty for topK=0, got %v", results)
	}
}

func TestRRFFuse_DedupWithinList(t *testing.T) {
	// 同一路中出现重复 ID，应只取首次排名
	dense := []core.SearchResult{
		{ID: "A", Content: "first", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
		{ID: "A", Content: "second", Score: 0.7}, // 重复
	}

	results := rrfFuse([][]core.SearchResult{dense}, 10)

	if len(results) != 2 {
		t.Fatalf("expected 2 results after dedup, got %d", len(results))
	}
	// A 的内容应该是第一次出现的
	if results[0].Content != "first" {
		t.Fatalf("expected content 'first' for A, got '%s'", results[0].Content)
	}
}

func TestRRFFuse_ScoreAccumulation(t *testing.T) {
	// doc_A 在 dense 排名 1，sparse 排名 2
	// doc_B 在 dense 排名 2，sparse 排名 1
	dense := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
	}
	sparse := []core.SearchResult{
		{ID: "B", Content: "B", Score: 0.5},
		{ID: "A", Content: "A", Score: 0.4},
	}

	results := rrfFuse([][]core.SearchResult{dense, sparse}, 10)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// A: 1/(60+1) + 1/(60+2) = 1/61 + 1/62
	// B: 1/(60+2) + 1/(60+1) = 1/62 + 1/61
	// A 和 B 的 RRF 分数应该相等
	scoreA := results[0].Score
	scoreB := results[1].Score
	if scoreA != scoreB {
		t.Fatalf("A and B should have equal RRF scores, got A=%f B=%f", scoreA, scoreB)
	}
}

func TestRRFFuse_ThreeLists(t *testing.T) {
	l1 := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
	}
	l2 := []core.SearchResult{
		{ID: "B", Content: "B", Score: 0.8},
	}
	l3 := []core.SearchResult{
		{ID: "C", Content: "C", Score: 0.7},
	}

	results := rrfFuse([][]core.SearchResult{l1, l2, l3}, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// 每个文档只在一路排名 1，RRF 分数相等
	for _, r := range results {
		expected := 1.0 / (float32(rrfK) + 1)
		if r.Score != expected {
			t.Fatalf("expected score %f for %s, got %f", expected, r.ID, r.Score)
		}
	}
}

func TestSortByScore(t *testing.T) {
	results := []core.SearchResult{
		{ID: "C", Content: "C", Score: 0.7},
		{ID: "A", Content: "A", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
	}

	sorted := sortByScore(results, 10)

	if len(sorted) != 3 {
		t.Fatalf("expected 3, got %d", len(sorted))
	}
	if sorted[0].ID != "A" || sorted[1].ID != "B" || sorted[2].ID != "C" {
		t.Fatalf("sort order mismatch: %v", sorted)
	}
}

func TestSortByScore_DedupAndTruncate(t *testing.T) {
	results := []core.SearchResult{
		{ID: "A", Content: "first", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
		{ID: "A", Content: "second", Score: 0.7},
		{ID: "C", Content: "C", Score: 0.6},
	}

	sorted := sortByScore(results, 2)

	if len(sorted) != 2 {
		t.Fatalf("expected 2, got %d", len(sorted))
	}
	if sorted[0].ID != "A" || sorted[0].Content != "first" {
		t.Fatalf("first result mismatch: %+v", sorted[0])
	}
	if sorted[1].ID != "B" {
		t.Fatalf("second result mismatch: %+v", sorted[1])
	}
}

func TestRRFFuse_MultiQueryMultiList(t *testing.T) {
	// 模拟 2 个子查询 × 2 种检索 = 4 条 ranked lists
	// q1-dense:  A(1), B(2)
	// q1-sparse: B(1), C(2)
	// q2-dense:  A(1), D(2)
	// q2-sparse: C(1), A(2)
	q1Dense := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.9},
		{ID: "B", Content: "B", Score: 0.8},
	}
	q1Sparse := []core.SearchResult{
		{ID: "B", Content: "B", Score: 0.5},
		{ID: "C", Content: "C", Score: 0.4},
	}
	q2Dense := []core.SearchResult{
		{ID: "A", Content: "A", Score: 0.85},
		{ID: "D", Content: "D", Score: 0.7},
	}
	q2Sparse := []core.SearchResult{
		{ID: "C", Content: "C", Score: 0.6},
		{ID: "A", Content: "A", Score: 0.3},
	}

	results := rrfFuse([][]core.SearchResult{q1Dense, q1Sparse, q2Dense, q2Sparse}, 10)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// A 出现在 q1-dense(rank1), q2-dense(rank1), q2-sparse(rank2) → 3 路命中
	// B 出现在 q1-dense(rank2), q1-sparse(rank1) → 2 路命中
	// C 出现在 q1-sparse(rank2), q2-sparse(rank1) → 2 路命中
	// D 只出现在 q2-dense(rank2) → 1 路命中

	// A 应该排第一（3 路命中）
	if results[0].ID != "A" {
		t.Fatalf("expected A at rank 1, got %s (score=%f)", results[0].ID, results[0].Score)
	}

	// B 和 C 都是 2 路命中
	// B: 1/(60+2) + 1/(60+1) = 1/62 + 1/61
	// C: 1/(60+2) + 1/(60+1) = 1/62 + 1/61
	// 同分，按 ID 排序 B < C
	if results[1].ID != "B" || results[2].ID != "C" {
		t.Fatalf("expected B then C at rank 2-3, got %s then %s", results[1].ID, results[2].ID)
	}

	// D 只有 1 路，排最后
	if results[3].ID != "D" {
		t.Fatalf("expected D at rank 4, got %s", results[3].ID)
	}

	// 验证 A 的 RRF 分数 = 1/(60+1) + 1/(60+1) + 1/(60+2) = 2/61 + 1/62
	expectedA := float32(2.0)/(float32(rrfK)+1) + 1.0/(float32(rrfK)+2)
	if results[0].Score != expectedA {
		t.Fatalf("A score: expected %f, got %f", expectedA, results[0].Score)
	}
}
