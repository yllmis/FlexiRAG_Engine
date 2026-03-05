package repository

import (
	"context"
	"errors"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent_mgmt"

	"gorm.io/gorm"
)

var _ core.AgentRepository = (*PGAgentRepo)(nil)

type PGAgentRepo struct {
	db *gorm.DB
}

func NewPGAgentRepo(db *gorm.DB) (*PGAgentRepo, error) {
	// 项目初期，自动建表
	err := db.AutoMigrate(&agent_mgmt.Agent{})
	if err != nil {
		return nil, err
	}
	return &PGAgentRepo{db: db}, nil
}

func (r *PGAgentRepo) Create(ctx context.Context, agent *agent_mgmt.Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

func (r *PGAgentRepo) GetByID(ctx context.Context, id uint) (*agent_mgmt.Agent, error) {
	var agent agent_mgmt.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不算是系统错误，返回 nil 即可
		}
		return nil, err
	}
	return &agent, nil
}
