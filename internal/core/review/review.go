package review

// Scope 审核范围
type Scope string

const (
	ScopeInput  Scope = "input"
	ScopeOutput Scope = "output"
	ScopeIngest Scope = "ingest"
)

// Action 审核动作
type Action string

const (
	ActionAllow  Action = "allow"
	ActionWarn   Action = "warn"
	ActionReview Action = "review"
	ActionReject Action = "reject"
)

// Severity 严重等级
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Result 审核结果
type Result struct {
	Passed       bool         `json:"passed"`
	Action       Action       `json:"action"`
	Severity     Severity     `json:"severity"`
	MatchedRules []MatchedRule `json:"matched_rules,omitempty"`
	Message      string       `json:"message"`
}

// MatchedRule 命中的规则摘要
type MatchedRule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Rule 审核规则定义
type Rule struct {
	ID        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	Pattern   string   `yaml:"pattern"`
	MatchType string   `yaml:"match_type"` // exact, contains, regex
	Category  string   `yaml:"category"`
	Action    Action   `yaml:"action"`
	Severity  Severity `yaml:"severity"`
	Scopes    []Scope  `yaml:"scopes"`
}

// RuleConfig 规则文件的顶层配置
type RuleConfig struct {
	Enabled bool           `yaml:"enabled"`
	Scopes  map[Scope]bool `yaml:"scopes"`
	Rules   []Rule         `yaml:"rules"`
}
