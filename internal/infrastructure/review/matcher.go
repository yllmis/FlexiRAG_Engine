package review

import (
	"context"
	"strings"

	domain "flexirag-engine/internal/core/review"
)

// 编译期接口实现检查
var _ domain.Reviewer = (*RuleReviewer)(nil)

// RuleReviewer 基于规则的审核器
type RuleReviewer struct {
	rules  []compiledRule
	scopes map[domain.Scope]bool // 全局 scope 开关
}

// NewRuleReviewer 创建规则审核器
// loadResult 为 nil 时，所有审核直接放行
func NewRuleReviewer(loadResult *LoadResult) *RuleReviewer {
	if loadResult == nil {
		return &RuleReviewer{}
	}
	return &RuleReviewer{
		rules:  loadResult.Rules,
		scopes: loadResult.Scopes,
	}
}

// severityWeight 用于比较 severity 高低
var severityWeight = map[domain.Severity]int{
	domain.SeverityLow:    1,
	domain.SeverityMedium: 2,
	domain.SeverityHigh:   3,
}

// actionWeight 用于同 severity 时比较动作优先级
var actionWeight = map[domain.Action]int{
	domain.ActionAllow:  0,
	domain.ActionWarn:   1,
	domain.ActionReview: 2,
	domain.ActionReject: 3,
}

func (r *RuleReviewer) Review(_ context.Context, scope domain.Scope, content string) (*domain.Result, error) {
	// 全局 scope 开关检查
	if r.scopes != nil && !r.scopes[scope] {
		return &domain.Result{
			Passed:  true,
			Action:  domain.ActionAllow,
			Message: string(scope) + " 审核已关闭",
		}, nil
	}

	if len(r.rules) == 0 {
		return &domain.Result{
			Passed:  true,
			Action:  domain.ActionAllow,
			Message: "无规则，直接放行",
		}, nil
	}

	var matched []domain.MatchedRule
	var topAction domain.Action
	var topSeverity domain.Severity

	for _, rule := range r.rules {
		if !rule.inScope(scope) {
			continue
		}
		if !rule.match(content) {
			continue
		}

		matched = append(matched, domain.MatchedRule{
			ID:   rule.ID,
			Name: rule.Name,
		})

		if isHigherPriority(rule.Severity, rule.Action, topSeverity, topAction) {
			topSeverity = rule.Severity
			topAction = rule.Action
		}
	}

	if len(matched) == 0 {
		return &domain.Result{
			Passed:  true,
			Action:  domain.ActionAllow,
			Message: "未命中任何规则",
		}, nil
	}

	return &domain.Result{
		// Passed 表示"是否继续处理"，只有 reject 中断流程
		// warn/review 都放行，由调用方根据 Action 决定后续行为
		Passed:       topAction != domain.ActionReject,
		Action:       topAction,
		Severity:     topSeverity,
		MatchedRules: matched,
		Message:      buildMessage(topAction, len(matched)),
	}, nil
}

// inScope 检查规则是否适用于当前审核范围
func (r *compiledRule) inScope(scope domain.Scope) bool {
	for _, s := range r.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// match 根据 match_type 执行匹配
func (r *compiledRule) match(content string) bool {
	switch r.MatchType {
	case "exact":
		return content == r.Pattern
	case "contains":
		return strings.Contains(content, r.Pattern)
	case "regex":
		return r.re != nil && r.re.MatchString(content)
	default:
		return false
	}
}

// isHigherPriority 判断新规则是否比当前最高优先级更高
func isHigherPriority(newSev domain.Severity, newAct domain.Action, curSev domain.Severity, curAct domain.Action) bool {
	newW := severityWeight[newSev]
	curW := severityWeight[curSev]
	if newW != curW {
		return newW > curW
	}
	return actionWeight[newAct] > actionWeight[curAct]
}

func buildMessage(action domain.Action, count int) string {
	switch action {
	case domain.ActionReject:
		return "命中高风险规则，已拦截"
	case domain.ActionReview:
		return "命中敏感规则，需人工复核"
	case domain.ActionWarn:
		return "命中规则，已记录警告"
	default:
		return "放行"
	}
}
