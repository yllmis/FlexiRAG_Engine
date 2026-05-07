package agent_mgmt

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrInvalidInput  = errors.New("invalid input")
)

type AgentService interface {
	Create(ctx context.Context, name, systemPrompt string) (*Agent, error)
	List(ctx context.Context) ([]Agent, error)
	GetByID(ctx context.Context, id uint) (*Agent, error)
	Update(ctx context.Context, id uint, name, systemPrompt *string) (*Agent, error)
	Delete(ctx context.Context, id uint) error
}

type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id uint) (*Agent, error)
	List(ctx context.Context) ([]Agent, error)
	Update(ctx context.Context, id uint, name, systemPrompt *string) (*Agent, error)
	Delete(ctx context.Context, id uint) (bool, error)
}

type agentService struct {
	repo AgentRepository
}

func NewAgentService(repo AgentRepository) AgentService {
	return &agentService{repo: repo}
}

func (s *agentService) Create(ctx context.Context, name, systemPrompt string) (*Agent, error) {
	name = strings.TrimSpace(name)
	systemPrompt = strings.TrimSpace(systemPrompt)
	if name == "" || systemPrompt == "" {
		return nil, ErrInvalidInput
	}

	agent := &Agent{Name: name, SystemPrompt: systemPrompt}
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *agentService) List(ctx context.Context) ([]Agent, error) {
	return s.repo.List(ctx)
}

func (s *agentService) GetByID(ctx context.Context, id uint) (*Agent, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

func (s *agentService) Update(ctx context.Context, id uint, name, systemPrompt *string) (*Agent, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var namePtr, promptPtr *string

	if name != nil {
		v := strings.TrimSpace(*name)
		if v == "" {
			return nil, ErrInvalidInput
		}
		namePtr = &v
	}
	if systemPrompt != nil {
		v := strings.TrimSpace(*systemPrompt)
		if v == "" {
			return nil, ErrInvalidInput
		}
		promptPtr = &v
	}
	if namePtr == nil && promptPtr == nil {
		return nil, ErrInvalidInput
	}

	agent, err := s.repo.Update(ctx, id, namePtr, promptPtr)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

func (s *agentService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidInput
	}
	ok, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAgentNotFound
	}
	return nil
}
