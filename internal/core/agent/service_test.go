package agent

import (
	"context"
	"errors"
	"testing"
)

type mockRepo struct {
	agents map[uint]*Agent
	nextID uint
	err    error
}

func newMockRepo() *mockRepo {
	return &mockRepo{agents: make(map[uint]*Agent), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, agent *Agent) error {
	if m.err != nil {
		return m.err
	}
	agent.ID = m.nextID
	m.agents[agent.ID] = agent
	m.nextID++
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uint) (*Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.agents[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func (m *mockRepo) List(_ context.Context) ([]Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []Agent
	for _, a := range m.agents {
		out = append(out, *a)
	}
	return out, nil
}

func (m *mockRepo) Update(_ context.Context, id uint, name, systemPrompt *string) (*Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.agents[id]
	if !ok {
		return nil, nil
	}
	if name != nil {
		a.Name = *name
	}
	if systemPrompt != nil {
		a.SystemPrompt = *systemPrompt
	}
	return a, nil
}

func (m *mockRepo) Delete(_ context.Context, id uint) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if _, ok := m.agents[id]; !ok {
		return false, nil
	}
	delete(m.agents, id)
	return true, nil
}

func TestService_Create_Success(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	agent, err := svc.Create(context.Background(), "test", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if agent.Name != "test" {
		t.Fatalf("expected name=test, got %s", agent.Name)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	_, err := svc.Create(context.Background(), "", "prompt")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	_, err = svc.Create(context.Background(), "test", "  ")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_Found(t *testing.T) {
	repo := newMockRepo()
	svc := NewAgentService(repo)
	repo.agents[1] = &Agent{ID: 1, Name: "a", SystemPrompt: "p"}

	agent, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != "a" {
		t.Fatalf("expected name=a, got %s", agent.Name)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	_, err := svc.GetByID(context.Background(), 99)
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	_, err := svc.GetByID(context.Background(), 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Update_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAgentService(repo)
	repo.agents[1] = &Agent{ID: 1, Name: "old", SystemPrompt: "old-p"}

	newName := "new"
	agent, err := svc.Update(context.Background(), 1, &newName, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != "new" {
		t.Fatalf("expected name=new, got %s", agent.Name)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	name := "x"
	_, err := svc.Update(context.Background(), 99, &name, nil)
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestService_Update_NoField(t *testing.T) {
	repo := newMockRepo()
	svc := NewAgentService(repo)
	repo.agents[1] = &Agent{ID: 1, Name: "a", SystemPrompt: "p"}

	_, err := svc.Update(context.Background(), 1, nil, nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Update_EmptyField(t *testing.T) {
	repo := newMockRepo()
	svc := NewAgentService(repo)
	repo.agents[1] = &Agent{ID: 1, Name: "a", SystemPrompt: "p"}

	empty := "   "
	_, err := svc.Update(context.Background(), 1, &empty, nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Delete_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAgentService(repo)
	repo.agents[1] = &Agent{ID: 1, Name: "a", SystemPrompt: "p"}

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.agents[1]; ok {
		t.Fatal("expected agent to be deleted")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	err := svc.Delete(context.Background(), 99)
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestService_Delete_InvalidID(t *testing.T) {
	svc := NewAgentService(newMockRepo())
	err := svc.Delete(context.Background(), 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.err = errors.New("db down")
	svc := NewAgentService(repo)

	_, err := svc.Create(context.Background(), "a", "p")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = svc.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	name := "x"
	_, err = svc.Update(context.Background(), 1, &name, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	err = svc.Delete(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
