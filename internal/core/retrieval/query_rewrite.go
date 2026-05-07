package retrieval

import "context"

// QueryRewriter 定义查询改写能力
// 将用户原始问题转化为一个或多个更适合检索的子查询
type QueryRewriter interface {
	// Rewrite 将原始问题改写为多个检索友好的子查询
	// 返回的 []string 至少包含原始问题本身（降级兜底）
	Rewrite(ctx context.Context, query string) ([]string, error)
}
