package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/ports"
	"flexirag-engine/internal/core/retrieval"
)

const defaultRewritePrompt = `你是一个搜索查询改写专家。请将用户的问题改写为适合向量检索的子查询。

严格遵守以下规则：
1. 【按需拆分】如果问题只包含 1 个意图，只输出 1 个改写后的查询，绝对不要为了凑数而生成同义句
2. 【多意图拆分】只有当问题明显包含多个不同话题时（用"和"、"以及"、"还有"等连接），才拆分为多个子查询
3. 【去口语化】去除口语化表达，补全省略的上下文，使用正式、完整的语句
4. 【数量限制】最多输出 %d 个子查询

请严格以 JSON 格式输出，不要添加任何其他内容：
{"sub_queries": ["子查询1", "子查询2"], "reasoning": "简要说明拆分逻辑"}

用户问题：%s`

// 编译期接口实现检查
var _ retrieval.QueryRewriter = (*LLMQueryRewriter)(nil)

type LLMQueryRewriter struct {
	llm        ports.LLMProvider
	maxQueries int
	fallback   bool // LLM 失败时是否降级返回原始问题
}

type RewriteOption func(*LLMQueryRewriter)

// WithMaxQueries 设置最大子查询数
func WithMaxQueries(n int) RewriteOption {
	return func(r *LLMQueryRewriter) {
		if n > 0 {
			r.maxQueries = n
		}
	}
}

// WithFallback 设置是否启用降级（LLM 失败时返回原始问题）
func WithFallback(enabled bool) RewriteOption {
	return func(r *LLMQueryRewriter) {
		r.fallback = enabled
	}
}

func NewLLMQueryRewriter(llm ports.LLMProvider, opts ...RewriteOption) *LLMQueryRewriter {
	r := &LLMQueryRewriter{
		llm:        llm,
		maxQueries: 3,
		fallback:   true,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// rewriteResult 是 LLM 返回的 JSON 结构
type rewriteResult struct {
	SubQueries []string `json:"sub_queries"`
	Reasoning  string   `json:"reasoning,omitempty"`
}

func (r *LLMQueryRewriter) Rewrite(ctx context.Context, query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []string{query}, nil
	}

	prompt := fmt.Sprintf(defaultRewritePrompt, r.maxQueries, query)
	messages := []core.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := r.llm.Chat(ctx, messages)
	if err != nil {
		if r.fallback {
			return []string{query}, nil
		}
		return nil, fmt.Errorf("查询改写失败: %w", err)
	}

	queries, reasoning := r.parseResponse(resp)

	if reasoning != "" {
		log.Printf("[QueryRewriter] 原始问题: %q, 改写逻辑: %s", query, reasoning)
	}

	if len(queries) == 0 {
		return []string{query}, nil
	}

	return queries, nil
}

// parseResponse 优先尝试 JSON 解析，失败则降级为文本行解析
func (r *LLMQueryRewriter) parseResponse(resp string) ([]string, string) {
	cleaned := stripCodeBlock(resp)

	// 尝试 JSON 解析
	var result rewriteResult
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil && len(result.SubQueries) > 0 {
		return dedup(result.SubQueries), result.Reasoning
	}

	// JSON 解析失败，降级为文本行解析
	return parseQueries(resp), ""
}

// stripCodeBlock 去除 LLM 常见的 markdown 代码块包装
// "```json\n{...}\n```" → "{...}"
func stripCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	// 剥离开头的 ```json 或 ```
	if strings.HasPrefix(s, "```") {
		firstLine := strings.IndexByte(s, '\n')
		if firstLine != -1 {
			s = s[firstLine+1:]
		}
		// 剥离结尾的 ```
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}

// dedup 去除重复字符串，保持顺序
func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

// parseQueries 解析 LLM 返回的多行文本为子查询列表（文本降级路径）
func parseQueries(resp string) []string {
	lines := strings.Split(resp, "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = removeListPrefix(line)
		if line != "" {
			queries = append(queries, line)
		}
	}
	return dedup(queries)
}

// removeListPrefix 去除列表编号前缀，如 "1. "、"2、"、"- "、"1）"
// 只在数字后紧跟分隔符时才去除，避免误伤 "2026年..." 这类内容
func removeListPrefix(line string) string {
	line = strings.TrimSpace(line)
	runes := []rune(line)
	if len(runes) == 0 {
		return ""
	}

	// 跳过前导符号前缀：- * •
	if runes[0] == '-' || runes[0] == '*' || runes[0] == '•' {
		return strings.TrimSpace(string(runes[1:]))
	}

	// 跳过数字 + 分隔符前缀
	digitEnd := 0
	for digitEnd < len(runes) && runes[digitEnd] >= '0' && runes[digitEnd] <= '9' {
		digitEnd++
	}
	if digitEnd > 0 && digitEnd < len(runes) {
		sep := runes[digitEnd]
		if sep == '.' || sep == '、' || sep == '）' || sep == ')' || sep == ' ' {
			return strings.TrimSpace(string(runes[digitEnd+1:]))
		}
	}

	return strings.TrimSpace(string(runes))
}
