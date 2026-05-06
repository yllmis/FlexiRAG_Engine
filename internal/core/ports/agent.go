package ports

import (
	"context"
	"flexirag-engine/internal/core/agent_mgmt"
)

// AgentRepository 定义 Agent 持久化操作标准
type AgentRepository interface {
	Create(ctx context.Context, agent *agent_mgmt.Agent) error
	GetByID(ctx context.Context, id uint) (*agent_mgmt.Agent, error)
	List(ctx context.Context) ([]agent_mgmt.Agent, error)
	Update(ctx context.Context, id uint, name, systemPrompt *string) (*agent_mgmt.Agent, error)
	Delete(ctx context.Context, id uint) (bool, error)
}