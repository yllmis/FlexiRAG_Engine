package review

import (
	"fmt"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"

	domain "flexirag-engine/internal/core/review"
)

// compiledRule 编译后的规则，正则已预编译
type compiledRule struct {
	domain.Rule
	re *regexp.Regexp // nil 表示非 regex 类型
}

// LoadRules 从 YAML 文件加载并编译审核规则
func LoadRules(path string) ([]compiledRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取规则文件失败: %w", err)
	}

	var cfg domain.RuleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析规则文件失败: %w", err)
	}

	if !cfg.Enabled {
		return nil, nil
	}

	rules := make([]compiledRule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		cr := compiledRule{Rule: rule}

		if rule.MatchType == "regex" {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, fmt.Errorf("规则 %s 正则编译失败: %w", rule.ID, err)
			}
			cr.re = re
		}

		rules = append(rules, cr)
	}

	return rules, nil
}
