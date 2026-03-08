package core

import (
	"context"
	"flexirag-engine/internal/core/agent_mgmt"
)

// ==========================================
// 核心数据结构定义
// ==========================================

// Message 代表 LLM 对话中的一条消息
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // "你是一个助教..."
}

// SearchResult 代表向量检索的结果
type SearchResult struct {
	ID      string  `json:"id"`
	Content string  `json:"content"` // 搜到的具体文本
	Score   float32 `json:"score"`   // 相似度 (0.0 - 1.0)
}

// ==========================================
// 核心接口定义 (Ports)
// ==========================================

// LLMProvider 定义了与大模型交互的标准
type LLMProvider interface {
	// Chat: 基础对话能力
	// ctx: 用于超时控制 (Timeout) 和取消 (Cancellation)
	Chat(ctx context.Context, messages []Message) (string, error)

	// Embed: 将文本转化为向量 (用于 RAG)
	// float32 比 float64 节省一半内存，且对向量检索精度足够
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// VectorStore 定义了向量数据库的操作标准
// 无论底层是 Milvus, Pinecone 还是 pgvector，都要实现这个接口
type VectorStore interface {
	// Upsert: 插入或更新向量
	// id: 数据的唯一标识
	// vector: 文本对应的向量
	// metadata: 存一些额外信息（比如这段话属于哪个 Agent，来源于哪个文件）
	Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error

	// Search: 根据向量搜索相似内容
	// topK: 返回前 K 个最相似的结果
	Search(ctx context.Context, agentId uint, vector []float32, topK int) ([]SearchResult, error)

	// Delete: 根据 ID 删除向量 (知识库更新时需要)
	Delete(ctx context.Context, id string) error
}

type AgentRepository interface {
	Create(ctx context.Context, agent *agent_mgmt.Agent) error

	GetByID(ctx context.Context, id uint) (*agent_mgmt.Agent, error)

	List(ctx context.Context) ([]agent_mgmt.Agent, error)

	Update(ctx context.Context, id uint, name, systemPrompt *string) (*agent_mgmt.Agent, error)
}
