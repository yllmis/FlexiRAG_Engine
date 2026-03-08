package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"flexirag-engine/internal/core/agent_mgmt"

	"github.com/gin-gonic/gin"
)

type testAPIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type mockAgentRepo struct {
	createFn             func(ctx context.Context, agent *agent_mgmt.Agent) error
	getByIDFn            func(ctx context.Context, id uint) (*agent_mgmt.Agent, error)
	listFn               func(ctx context.Context) ([]agent_mgmt.Agent, error)
	updateSystemPromptFn func(ctx context.Context, id uint, systemPrompt string) (*agent_mgmt.Agent, error)
}

func (m *mockAgentRepo) Create(ctx context.Context, agent *agent_mgmt.Agent) error {
	if m.createFn != nil {
		return m.createFn(ctx, agent)
	}
	return nil
}

func (m *mockAgentRepo) GetByID(ctx context.Context, id uint) (*agent_mgmt.Agent, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockAgentRepo) List(ctx context.Context) ([]agent_mgmt.Agent, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockAgentRepo) UpdateSystemPrompt(ctx context.Context, id uint, systemPrompt string) (*agent_mgmt.Agent, error) {
	if m.updateSystemPromptFn != nil {
		return m.updateSystemPromptFn(ctx, id, systemPrompt)
	}
	return nil, nil
}

func TestUpdateAgentSystemPrompt_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{
		updateSystemPromptFn: func(ctx context.Context, id uint, systemPrompt string) (*agent_mgmt.Agent, error) {
			return &agent_mgmt.Agent{ID: id, Name: "测试Agent", SystemPrompt: systemPrompt}, nil
		},
	})

	r := gin.New()
	r.PATCH("/api/v1/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/1/system-prompt", strings.NewReader(`{"system_prompt":"新提示词"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if got := resp["code"]; got != float64(http.StatusOK) {
		t.Fatalf("期望 code=200，实际=%v", got)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("响应 data 不是对象：%v", resp["data"])
	}
	if got := data["system_prompt"]; got != "新提示词" {
		t.Fatalf("期望 system_prompt=新提示词，实际=%v", got)
	}
}

func TestUpdateAgentSystemPrompt_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{})
	r := gin.New()
	r.PATCH("/api/v1/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/abc/system-prompt", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	var resp testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("期望业务码 400，实际 %d", resp.Code)
	}
}

func TestUpdateAgentSystemPrompt_EmptyPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{})
	r := gin.New()
	r.PATCH("/api/v1/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/1/system-prompt", strings.NewReader(`{"system_prompt":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgentSystemPrompt_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{
		updateSystemPromptFn: func(ctx context.Context, id uint, systemPrompt string) (*agent_mgmt.Agent, error) {
			return nil, nil
		},
	})

	r := gin.New()
	r.PATCH("/api/v1/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/1/system-prompt", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望状态码 404，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgentSystemPrompt_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{
		updateSystemPromptFn: func(ctx context.Context, id uint, systemPrompt string) (*agent_mgmt.Agent, error) {
			return nil, errors.New("db error")
		},
	})

	r := gin.New()
	r.PATCH("/api/v1/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/1/system-prompt", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望状态码 500，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestListAgents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{
		listFn: func(ctx context.Context) ([]agent_mgmt.Agent, error) {
			return []agent_mgmt.Agent{
				{ID: 1, Name: "AgentA", SystemPrompt: "PromptA"},
				{ID: 2, Name: "AgentB", SystemPrompt: "PromptB"},
			}, nil
		},
	})

	r := gin.New()
	r.GET("/api/v1/agents", h.ListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	var resp testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("期望业务码 200，实际 %d", resp.Code)
	}

	var data struct {
		Agents []agent_mgmt.Agent `json:"agents"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(data.Agents) != 2 {
		t.Fatalf("期望返回 2 个 Agent，实际 %d", len(data.Agents))
	}
}

func TestListAgents_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentRepo{
		listFn: func(ctx context.Context) ([]agent_mgmt.Agent, error) {
			return nil, errors.New("db error")
		},
	})

	r := gin.New()
	r.GET("/api/v1/agents", h.ListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望状态码 500，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	var resp testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("期望业务码 500，实际 %d", resp.Code)
	}
}
