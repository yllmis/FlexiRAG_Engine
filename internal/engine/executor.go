package engine

import (
	"context"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent_mgmt"

	"fmt"
	"strings"
)

type AgentEngine struct {
	llm    core.LLMProvider
	vector core.VectorStore
}

func NewAgentEngine(llm core.LLMProvider, vector core.VectorStore) *AgentEngine {
	return &AgentEngine{
		llm:    llm,
		vector: vector,
	}
}

func (e *AgentEngine) ProcessQuery(ctx context.Context, agent *agent_mgmt.Agent, query string) (string, error) {
	vectors, err := e.llm.Embed(ctx, []string{query})
	if err != nil {
		return "", fmt.Errorf("生成向量失败: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return "", fmt.Errorf("生成向量失败: 返回空向量")
	}

	searchResults, err := e.vector.Search(ctx, agent.ID, vectors[0], 3)
	if err != nil {
		return "", fmt.Errorf("向量搜索失败: %w", err)
	}

	var contextBuilder strings.Builder
	for i, result := range searchResults {
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
	// 动态设置智能体角色
	systemPrompt := agent.SystemPrompt
	if systemPrompt == "" {
		// 如果用户没配置，给一个默认的兜底
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
