package review

import "context"

// Reviewer 定义审核能力
// 对输入/输出/入库内容执行规则检查，返回审核结果
type Reviewer interface {
	// Review 对给定内容执行审核
	// scope: 审核范围（input/output/ingest）
	// content: 待审核的文本内容
	Review(ctx context.Context, scope Scope, content string) (*Result, error)
}
