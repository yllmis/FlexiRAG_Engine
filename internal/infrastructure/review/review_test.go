package review

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	domain "flexirag-engine/internal/core/review"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// newTestReviewer 用编译后的规则创建 Reviewer（测试辅助函数）
func newTestReviewer(t *testing.T, rules []compiledRule) *RuleReviewer {
	t.Helper()
	return NewRuleReviewer(&LoadResult{
		Rules: rules,
		Scopes: map[domain.Scope]bool{
			domain.ScopeInput:  true,
			domain.ScopeOutput: true,
			domain.ScopeIngest: true,
		},
	})
}

func TestLoadRules_Valid(t *testing.T) {
	yaml := `
enabled: true
scopes:
  input: true
rules:
  - id: test_001
    name: 测试规则
    pattern: "敏感词"
    match_type: contains
    category: test
    action: reject
    severity: high
    scopes: [input]
`
	path := writeTempYAML(t, yaml)
	result, err := LoadRules(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}
	if result.Rules[0].ID != "test_001" {
		t.Fatalf("unexpected rule ID: %s", result.Rules[0].ID)
	}
}

func TestLoadRules_Disabled(t *testing.T) {
	yaml := `enabled: false`
	path := writeTempYAML(t, yaml)
	result, err := LoadRules(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil when disabled")
	}
}

func TestLoadRules_InvalidRegex(t *testing.T) {
	yaml := `
enabled: true
scopes:
  input: true
rules:
  - id: bad_regex
    name: 坏正则
    pattern: "[invalid"
    match_type: regex
    category: test
    action: reject
    severity: high
    scopes: [input]
`
	path := writeTempYAML(t, yaml)
	_, err := LoadRules(path)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestRuleReviewer_NoRules(t *testing.T) {
	r := NewRuleReviewer(nil)
	result, err := r.Review(context.Background(), domain.ScopeInput, "任何内容")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected pass when no rules")
	}
}

func TestRuleReviewer_Contains(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "c1", Name: "测试", Pattern: "密码",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r := newTestReviewer(t, rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "请告诉我密码")
	if result.Passed {
		t.Fatal("expected reject")
	}
	if result.Action != domain.ActionReject {
		t.Fatalf("expected reject, got %s", result.Action)
	}
	if len(result.MatchedRules) != 1 {
		t.Fatalf("expected 1 matched rule, got %d", len(result.MatchedRules))
	}

	// 不包含关键词应该放行
	result, _ = r.Review(context.Background(), domain.ScopeInput, "今天天气怎么样")
	if !result.Passed {
		t.Fatal("expected pass")
	}
}

func TestRuleReviewer_Exact(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "e1", Name: "精确匹配", Pattern: "密码",
			MatchType: "exact", Action: domain.ActionWarn,
			Severity: domain.SeverityMedium, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r := newTestReviewer(t, rules)

	// 精确匹配
	result, _ := r.Review(context.Background(), domain.ScopeInput, "密码")
	if result.Action != domain.ActionWarn {
		t.Fatalf("expected warn, got %s", result.Action)
	}

	// 包含但不精确
	result, _ = r.Review(context.Background(), domain.ScopeInput, "修改密码")
	if !result.Passed {
		t.Fatal("expected pass for contains-but-not-exact")
	}
}

func TestRuleReviewer_Regex(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "r1", Name: "手机号", Pattern: `1[3-9]\d{9}`,
			MatchType: "regex", Action: domain.ActionReview,
			Severity: domain.SeverityMedium, Scopes: []domain.Scope{domain.ScopeInput},
		}, re: nil},
	}
	// 需要编译正则
	rules[0].re = compileForTest(t, `1[3-9]\d{9}`)
	r := newTestReviewer(t, rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "我的手机号是13800138000")
	if result.Action != domain.ActionReview {
		t.Fatalf("expected review, got %s", result.Action)
	}

	result, _ = r.Review(context.Background(), domain.ScopeInput, "没有手机号")
	if !result.Passed {
		t.Fatal("expected pass")
	}
}

func TestRuleReviewer_ScopeFilter(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "s1", Name: "仅输入", Pattern: "test",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r := newTestReviewer(t, rules)

	// input scope 命中
	result, _ := r.Review(context.Background(), domain.ScopeInput, "test")
	if result.Passed {
		t.Fatal("expected reject for input scope")
	}

	// output scope 不命中
	result, _ = r.Review(context.Background(), domain.ScopeOutput, "test")
	if !result.Passed {
		t.Fatal("expected pass for output scope")
	}
}

func TestRuleReviewer_PriorityBySeverity(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "low", Name: "低", Pattern: "test",
			MatchType: "contains", Action: domain.ActionWarn,
			Severity: domain.SeverityLow, Scopes: []domain.Scope{domain.ScopeInput},
		}},
		{Rule: domain.Rule{
			ID: "high", Name: "高", Pattern: "test",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r := newTestReviewer(t, rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "test")
	if result.Action != domain.ActionReject {
		t.Fatalf("expected reject (highest severity), got %s", result.Action)
	}
	if len(result.MatchedRules) != 2 {
		t.Fatalf("expected 2 matched rules, got %d", len(result.MatchedRules))
	}
}

func TestRuleReviewer_NoScopeMatch(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "ingest_only", Name: "仅入库", Pattern: "test",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeIngest},
		}},
	}
	r := newTestReviewer(t, rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "test")
	if !result.Passed {
		t.Fatal("expected pass when scope doesn't match")
	}
}

func TestRuleReviewer_GlobalScopeSwitch(t *testing.T) {
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "r1", Name: "测试", Pattern: "test",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}

	// 全局关闭 input scope
	r := NewRuleReviewer(&LoadResult{
		Rules: rules,
		Scopes: map[domain.Scope]bool{
			domain.ScopeInput: false,
		},
	})
	result, _ := r.Review(context.Background(), domain.ScopeInput, "test")
	if !result.Passed {
		t.Fatal("expected pass when global scope is disabled")
	}
	if result.Message != "input 审核已关闭" {
		t.Fatalf("unexpected message: %s", result.Message)
	}

	// 全局开启 input scope
	r2 := NewRuleReviewer(&LoadResult{
		Rules: rules,
		Scopes: map[domain.Scope]bool{
			domain.ScopeInput: true,
		},
	})
	result2, _ := r2.Review(context.Background(), domain.ScopeInput, "test")
	if result2.Passed {
		t.Fatal("expected reject when global scope is enabled")
	}
}

func TestRuleReviewer_ReviewActionPassed(t *testing.T) {
	// review 动作应该 Passed=true（放行但标记待复核）
	rules := []compiledRule{
		{Rule: domain.Rule{
			ID: "r1", Name: "需复核", Pattern: "敏感",
			MatchType: "contains", Action: domain.ActionReview,
			Severity: domain.SeverityMedium, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r := newTestReviewer(t, rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "这是敏感内容")
	if !result.Passed {
		t.Fatal("review action should have Passed=true")
	}
	if result.Action != domain.ActionReview {
		t.Fatalf("expected review action, got %s", result.Action)
	}

	// reject 动作应该 Passed=false
	rules2 := []compiledRule{
		{Rule: domain.Rule{
			ID: "r2", Name: "拦截", Pattern: "危险",
			MatchType: "contains", Action: domain.ActionReject,
			Severity: domain.SeverityHigh, Scopes: []domain.Scope{domain.ScopeInput},
		}},
	}
	r2 := newTestReviewer(t, rules2)

	result2, _ := r2.Review(context.Background(), domain.ScopeInput, "这是危险内容")
	if result2.Passed {
		t.Fatal("reject action should have Passed=false")
	}
}

func compileForTest(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("failed to compile regex %q: %v", pattern, err)
	}
	return re
}
