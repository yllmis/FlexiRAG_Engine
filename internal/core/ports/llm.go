package ports

import (
	"context"
	"flexirag-engine/internal/core"
)

// LLMProvider 定义了与大模型交互的标准
type LLMProvider interface {
	// Chat: 基础对话能力
	Chat(ctx context.Context, messages []core.Message) (string, error)

	// Embed: 将文本转化为向量 (用于 RAG)
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}