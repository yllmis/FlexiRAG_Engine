package llm

import (
	"context"
	"flexirag-engine/internal/core"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// 编译期接口实现检查
var _ core.LLMProvider = (*OpenAIClient)(nil)

type OpenAIClient struct {
	client     *openai.Client
	chatModel  string                // 比如 "gpt-4o-mini" (性价比极高，适合测试)
	embedModel openai.EmbeddingModel // 比如 "text-embedding-3-small"
}

// NewOpenAIClient 工厂方法
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		client:     openai.NewClient(apiKey),
		chatModel:  openai.GPT4oMini,       // 默认对话模型
		embedModel: openai.SmallEmbedding3, // 默认向量模型 (1536维)
	}
}

// Chat 对话方法实现
func (c *OpenAIClient) Chat(ctx context.Context, messages []core.Message) (string, error) {
	// 1. 组装 OpenAI 格式的 Message
	var oaiMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 2. 发起网络请求
	req := openai.ChatCompletionRequest{
		Model:       c.chatModel,
		Messages:    oaiMessages,
		Temperature: 0.7,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI Chat API 调用失败: %w", err)
	}

	// 3. 返回结果
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("OpenAI 返回了空结果")
}

// Embed 向量化方法实现 (重点来了！)
func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// 1. 发起批量向量化请求
	req := openai.EmbeddingRequest{
		Input: texts, // 直接把 []string 塞进去，API 原生支持批量！
		Model: c.embedModel,
	}

	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI Embed API 调用失败: %w", err)
	}

	// 2. 提取并组装我们需要的 [][]float32 格式
	// 预分配内存，提升性能 (资深 Go 工程师习惯)
	result := make([][]float32, len(texts))

	// OpenAI 返回的 resp.Data 是一个数组，每个元素包含一个 Embedding 切片
	for i, data := range resp.Data {
		result[i] = data.Embedding
	}

	return result, nil
}
