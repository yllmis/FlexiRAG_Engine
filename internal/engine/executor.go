package engine

import (
	"context"
	"errors"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent"
	"flexirag-engine/internal/core/ports"
	"flexirag-engine/internal/core/retrieval"
	"flexirag-engine/internal/core/review"
	"fmt"
	"sort"
	"strings"
)

// ErrReviewRejected 输入审核被拒绝时返回
var ErrReviewRejected = errors.New("输入审核未通过")

type AgentEngine struct {
	llm      ports.LLMProvider
	vector   ports.VectorStore
	rewriter retrieval.QueryRewriter // 可选，nil 表示不做改写
	reviewer review.Reviewer         // 可选，nil 表示不做审核
}

type EngineOption func(*AgentEngine)

func WithQueryRewriter(r retrieval.QueryRewriter) EngineOption {
	return func(e *AgentEngine) {
		e.rewriter = r
	}
}

func WithReviewer(r review.Reviewer) EngineOption {
	return func(e *AgentEngine) {
		e.reviewer = r
	}
}

func NewAgentEngine(llm ports.LLMProvider, vector ports.VectorStore, opts ...EngineOption) *AgentEngine {
	e := &AgentEngine{
		llm:    llm,
		vector: vector,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

const defaultTopK = 3

func (e *AgentEngine) ProcessQuery(ctx context.Context, agt *agent.Agent, query string) (string, error) {
	// Step 0: 输入审核（可选）
	if e.reviewer != nil {
		result, err := e.reviewer.Review(ctx, review.ScopeInput, query)
		if err != nil {
			return "", fmt.Errorf("输入审核异常: %w", err)
		}
		if !result.Passed {
			return "", fmt.Errorf("%w: %s", ErrReviewRejected, result.Message)
		}
	}

	// Step 1: 查询改写（可选）
	queries := []string{query}
	if e.rewriter != nil {
		rewritten, err := e.rewriter.Rewrite(ctx, query)
		if err == nil && len(rewritten) > 0 {
			queries = rewritten
		}
	}

	// Step 2: 对每个子查询分别 Embedding + 检索，合并去重
	allResults := e.retrieveAll(ctx, agt.ID, queries)

	// Step 3: 拼接 Context
	var contextBuilder strings.Builder
	for i, result := range allResults {
		if strings.TrimSpace(result.Content) == "" {
			continue
		}
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, result.Content))
	}
	contextInfo := strings.TrimSpace(contextBuilder.String())

	userPrompt := fmt.Sprintf(
		"请严格依据以下 <context> 标签内的信息回答我的问题。\n\n<context>\n%s\n</context>\n\n我的问题是：%s",
		contextInfo,
		query,
	)

	systemPrompt := agt.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个智能助手，请依据提供的上下文进行客观回答。如果上下文中没有相关信息，请明确告知用户，不要捏造事实。"
	}

	messages := []core.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := e.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("调用LLM生成回答失败: %w", err)
	}

	return response, nil
}

// retrieveAll 对多个子查询分别检索，合并去重后按 Score 降序取 Top-K
func (e *AgentEngine) retrieveAll(ctx context.Context, agentID uint, queries []string) []core.SearchResult {
	seen := make(map[string]bool)
	var merged []core.SearchResult

	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}

		vectors, err := e.llm.Embed(ctx, []string{q})
		if err != nil || len(vectors) == 0 || len(vectors[0]) == 0 {
			continue
		}

		results, err := e.vector.Search(ctx, agentID, vectors[0], defaultTopK)
		if err != nil {
			continue
		}

		for _, r := range results {
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			merged = append(merged, r)
		}
	}

	// 按 Score 降序排列，取前 defaultTopK
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	if len(merged) > defaultTopK {
		merged = merged[:defaultTopK]
	}

	return merged
}
