package llm

import (
	"context"
	"flexirag-engine/internal/core"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	defaultGLMBaseURL    = "https://open.bigmodel.cn/api/paas/v4/"
	defaultGLMChatModel  = "glm-4-flash"
	defaultGLMEmbedModel = openai.EmbeddingModel("embedding-3")
)

// 编译期接口实现检查
var _ core.LLMProvider = (*GLMClient)(nil)

type GLMClient struct {
	client     *openai.Client
	chatModel  string
	embedModel openai.EmbeddingModel
}

// NewGLMClient 使用默认 GLM 配置创建客户端。
func NewGLMClient(apiKey string) *GLMClient {
	return NewGLMClientWithConfig(apiKey, defaultGLMBaseURL, defaultGLMChatModel, defaultGLMEmbedModel)
}

// NewGLMClientWithConfig 支持自定义 BaseURL 与模型名。
func NewGLMClientWithConfig(apiKey, baseURL, chatModel string, embedModel openai.EmbeddingModel) *GLMClient {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = strings.TrimRight(baseURL, "/") + "/"

	if strings.TrimSpace(chatModel) == "" {
		chatModel = defaultGLMChatModel
	}
	if strings.TrimSpace(string(embedModel)) == "" {
		embedModel = defaultGLMEmbedModel
	}

	return &GLMClient{
		client:     openai.NewClientWithConfig(cfg),
		chatModel:  chatModel,
		embedModel: embedModel,
	}
}

// Chat 对话方法实现
func (c *GLMClient) Chat(ctx context.Context, messages []core.Message) (string, error) {
	oaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       c.chatModel,
		Messages:    oaiMessages,
		Temperature: 0.7,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GLM Chat API 调用失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("GLM 返回了空结果")
	}

	return resp.Choices[0].Message.Content, nil
}

// Embed 向量化方法实现
func (c *GLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: c.embedModel,
	}

	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("GLM Embed API 调用失败: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("GLM Embedding 返回空结果")
	}

	result := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		result[i] = data.Embedding
	}
	return result, nil
}
