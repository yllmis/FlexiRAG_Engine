package repository

import (
	"context"
	"errors"
	"flexirag-engine/internal/core/agent"

	"gorm.io/gorm"
)

var _ agent.AgentRepository = (*PGAgentRepo)(nil)

type PGAgentRepo struct {
	db *gorm.DB
}

func NewPGAgentRepo(db *gorm.DB) (*PGAgentRepo, error) {
	// 项目初期，自动建表
	err := db.AutoMigrate(&agent.Agent{})
	if err != nil {
		return nil, err
	}
	return &PGAgentRepo{db: db}, nil
}

func (r *PGAgentRepo) Create(ctx context.Context, agent *agent.Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

func (r *PGAgentRepo) GetByID(ctx context.Context, id uint) (*agent.Agent, error) {
	var agent agent.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不算是系统错误，返回 nil 即可
		}
		return nil, err
	}
	return &agent, nil
}

func (r *PGAgentRepo) List(ctx context.Context) ([]agent.Agent, error) {
	var agents []agent.Agent
	err := r.db.WithContext(ctx).Order("id ASC").Find(&agents).Error
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (r *PGAgentRepo) Update(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
	updates := map[string]interface{}{}
	if name != nil {
		updates["name"] = *name
	}
	if systemPrompt != nil {
		updates["system_prompt"] = *systemPrompt
	}
	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	result := r.db.WithContext(ctx).
		Model(&agent.Agent{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	return r.GetByID(ctx, id)
}

func (r *PGAgentRepo) Delete(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Delete(&agent.Agent{}, id)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
