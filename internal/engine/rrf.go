package engine

import (
	"flexirag-engine/internal/core"
	"sort"
)

// rrfK 是 RRF 算法的常数参数，来自原始论文（Cormack et al., 2009）
// k=60 的设计意图：抑制排名靠前的结果获得过大的分数优势
// Top-1 贡献 1/61 ≈ 0.0164，Top-2 贡献 1/62 ≈ 0.0161，差距仅 2%
const rrfK = 60

// rrfFuse 对多路检索结果做 Reciprocal Rank Fusion
//
//	算法：RRF_score(d) = Σ 1/(k + rank_i(d))
//	- 用排名代替原始分数，消除不同检索路径的分数量纲差异
//	- 同一文档在多路中都命中时，RRF 分数叠加
//
// 输入：多组已按相关性排序的结果列表（每组内部排名即为数组顺序）
// 输出：融合后按 RRF 分数降序排列的结果，取 topK
func rrfFuse(lists [][]core.SearchResult, topK int) []core.SearchResult {
	if topK <= 0 {
		return nil
	}

	scores := make(map[string]float32)  // ID → RRF 累计分数
	content := make(map[string]string)  // ID → Content（保留首次出现的内容）

	for _, list := range lists {
		for rank, r := range list {
			// 同一路中重复出现的 ID，只取首次排名
			if _, exists := scores[r.ID]; !exists {
				content[r.ID] = r.Content
			}
			scores[r.ID] += 1.0 / (float32(rrfK) + float32(rank+1))
		}
	}

	results := make([]core.SearchResult, 0, len(scores))
	for id, score := range scores {
		results = append(results, core.SearchResult{
			ID:      id,
			Content: content[id],
			Score:   score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID // 同分时按 ID 稳定排序
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

// sortByScore 按原始 Score 降序排列并截取 topK
// 供 sparse 未启用时的纯 dense 退化路径使用
func sortByScore(results []core.SearchResult, topK int) []core.SearchResult {
	if topK <= 0 {
		return nil
	}

	// 去重：同一路检索结果中可能出现重复 ID（多子查询场景）
	seen := make(map[string]bool)
	deduped := make([]core.SearchResult, 0, len(results))
	for _, r := range results {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		deduped = append(deduped, r)
	}

	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Score == deduped[j].Score {
			return deduped[i].ID < deduped[j].ID
		}
		return deduped[i].Score > deduped[j].Score
	})

	if len(deduped) > topK {
		deduped = deduped[:topK]
	}
	return deduped
}
