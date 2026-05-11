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
	rules, err := LoadRules(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "test_001" {
		t.Fatalf("unexpected rule ID: %s", rules[0].ID)
	}
}

func TestLoadRules_Disabled(t *testing.T) {
	yaml := `enabled: false`
	path := writeTempYAML(t, yaml)
	rules, err := LoadRules(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Fatalf("expected nil rules when disabled, got %d", len(rules))
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
	r := NewRuleReviewer(rules)

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
	r := NewRuleReviewer(rules)

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
	r := NewRuleReviewer(rules)

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
	r := NewRuleReviewer(rules)

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
	r := NewRuleReviewer(rules)

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
	r := NewRuleReviewer(rules)

	result, _ := r.Review(context.Background(), domain.ScopeInput, "test")
	if !result.Passed {
		t.Fatal("expected pass when scope doesn't match")
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
