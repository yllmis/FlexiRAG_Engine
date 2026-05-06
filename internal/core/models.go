package core

import "time"

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

// Subject 代表鉴权后的调用主体
type Subject struct {
	ID string `json:"id"`
}

// AuditEvent 代表一条审计事件
type AuditEvent struct {
	EventType    string    `json:"event_type"`
	SubjectID    string    `json:"subject_id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	RequestID    string    `json:"request_id"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}