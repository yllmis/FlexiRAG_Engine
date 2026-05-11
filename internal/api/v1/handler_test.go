package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/core/ports"
	"flexirag-engine/internal/core/review"
	"flexirag-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

type testAPIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type mockAuditLogger struct {
	events []core.AuditEvent
}

func (m *mockAuditLogger) Log(event core.AuditEvent) {
	m.events = append(m.events, event)
}

type testLLM struct {
	chatResp string
}

func (m *testLLM) Chat(ctx context.Context, messages []core.Message) (string, error) {
	return m.chatResp, nil
}

func (m *testLLM) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2}}, nil
}

type testVector struct{}

func (m *testVector) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	return nil
}

func (m *testVector) Search(ctx context.Context, agentID uint, vector []float32, topK int) ([]core.SearchResult, error) {
	return []core.SearchResult{
		{ID: "c1", Content: "知识片段", Score: 0.9},
	}, nil
}

func (m *testVector) Delete(ctx context.Context, id string) error {
	return nil
}

type testReviewer struct {
	fn func(ctx context.Context, scope review.Scope, content string) (*review.Result, error)
}

func (m *testReviewer) Review(ctx context.Context, scope review.Scope, content string) (*review.Result, error) {
	return m.fn(ctx, scope, content)
}

var _ ports.LLMProvider = (*testLLM)(nil)
var _ ports.VectorStore = (*testVector)(nil)
var _ review.Reviewer = (*testReviewer)(nil)

type mockAgentSvc struct {
	createFn  func(ctx context.Context, name, systemPrompt string) (*agent.Agent, error)
	listFn    func(ctx context.Context) ([]agent.Agent, error)
	getByIDFn func(ctx context.Context, id uint) (*agent.Agent, error)
	updateFn  func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error)
	deleteFn  func(ctx context.Context, id uint) error
}

func (m *mockAgentSvc) Create(ctx context.Context, name, systemPrompt string) (*agent.Agent, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, systemPrompt)
	}
	return &agent.Agent{ID: 1, Name: name, SystemPrompt: systemPrompt}, nil
}

func (m *mockAgentSvc) List(ctx context.Context) ([]agent.Agent, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockAgentSvc) GetByID(ctx context.Context, id uint) (*agent.Agent, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, agent.ErrAgentNotFound
}

func (m *mockAgentSvc) Update(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, name, systemPrompt)
	}
	return nil, nil
}

func (m *mockAgentSvc) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func TestUpdateAgent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		updateFn: func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
			updatedName := "测试Agent"
			updatedPrompt := "旧提示词"
			if name != nil {
				updatedName = *name
			}
			if systemPrompt != nil {
				updatedPrompt = *systemPrompt
			}
			return &agent.Agent{ID: id, Name: updatedName, SystemPrompt: updatedPrompt}, nil
		},
	})

	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/1", strings.NewReader(`{"name":"新名称","system_prompt":"新提示词"}`))
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
	if got := data["name"]; got != "新名称" {
		t.Fatalf("期望 name=新名称，实际=%v", got)
	}
}

func TestUpdateAgent_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{})
	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/abc", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_EmptyPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		updateFn: func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
			return nil, agent.ErrInvalidInput
		},
	})
	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/1", strings.NewReader(`{"system_prompt":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_NoField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		updateFn: func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
			return nil, agent.ErrInvalidInput
		},
	})
	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		updateFn: func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
			return nil, agent.ErrAgentNotFound
		},
	})

	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/1", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望状态码 404，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		updateFn: func(ctx context.Context, id uint, name, systemPrompt *string) (*agent.Agent, error) {
			return nil, errors.New("db error")
		},
	})

	r := gin.New()
	r.PUT("/api/v1/agents/:id", h.UpdateAgent)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/1", strings.NewReader(`{"system_prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望状态码 500，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestListAgents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		listFn: func(ctx context.Context) ([]agent.Agent, error) {
			return []agent.Agent{
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
		Agents []agent.Agent `json:"agents"`
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

	h := NewHandler(nil, nil, &mockAgentSvc{
		listFn: func(ctx context.Context) ([]agent.Agent, error) {
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
}

func TestDeleteAgent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		deleteFn: func(ctx context.Context, id uint) error {
			if id == 1 {
				return nil
			}
			return agent.ErrAgentNotFound
		},
	})

	r := gin.New()
	r.DELETE("/api/v1/agents/:id", h.DeleteAgent)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestDeleteAgent_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{})
	r := gin.New()
	r.DELETE("/api/v1/agents/:id", h.DeleteAgent)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, nil, &mockAgentSvc{
		deleteFn: func(ctx context.Context, id uint) error {
			return agent.ErrAgentNotFound
		},
	})
	r := gin.New()
	r.DELETE("/api/v1/agents/:id", h.DeleteAgent)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望状态码 404，实际 %d，响应：%s", w.Code, w.Body.String())
	}
}

func TestChat_OutputRejectMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reviewer := &testReviewer{
		fn: func(ctx context.Context, scope review.Scope, content string) (*review.Result, error) {
			switch scope {
			case review.ScopeInput:
				return &review.Result{Passed: true, Action: review.ActionAllow}, nil
			case review.ScopeOutput:
				return &review.Result{
					Passed:  false,
					Action:  review.ActionReject,
					Message: "输出命中高风险规则",
				}, nil
			default:
				return &review.Result{Passed: true, Action: review.ActionAllow}, nil
			}
		},
	}
	engineSvc := engine.NewAgentEngine(&testLLM{chatResp: "回答内容"}, &testVector{}, engine.WithReviewer(reviewer))
	h := NewHandler(engineSvc, knowledge.NewChunkService(&testLLM{}, &testVector{}), &mockAgentSvc{
		getByIDFn: func(ctx context.Context, id uint) (*agent.Agent, error) {
			return &agent.Agent{ID: id, Name: "A", SystemPrompt: "P"}, nil
		},
	})

	r := gin.New()
	r.POST("/api/v1/chat", h.Chat)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(`{"query":"正常问题","agent_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望状态码 403，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	var resp testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Msg != "输出内容未通过审核" {
		t.Fatalf("期望输出审核提示，实际=%s", resp.Msg)
	}
}

func TestIngestKnowledge_ReviewAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auditLogger := &mockAuditLogger{}
	reviewer := &testReviewer{
		fn: func(ctx context.Context, scope review.Scope, content string) (*review.Result, error) {
			if scope == review.ScopeIngest {
				return &review.Result{
					Passed:  true,
					Action:  review.ActionReview,
					Message: "命中敏感规则，需人工复核",
				}, nil
			}
			return &review.Result{Passed: true, Action: review.ActionAllow}, nil
		},
	}
	llm := &testLLM{}
	vector := &testVector{}
	engineSvc := engine.NewAgentEngine(llm, vector, engine.WithReviewer(reviewer))
	h := NewHandler(engineSvc, knowledge.NewChunkService(llm, vector), &mockAgentSvc{
		getByIDFn: func(ctx context.Context, id uint) (*agent.Agent, error) {
			return &agent.Agent{ID: id, Name: "A", SystemPrompt: "P"}, nil
		},
	}, auditLogger)

	r := gin.New()
	r.POST("/api/v1/knowledge/ingest", h.IngestKnowledge)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", strings.NewReader(`{"text":"用于测试的知识文本","agent_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际 %d，响应：%s", w.Code, w.Body.String())
	}

	foundReview := false
	foundSuccess := false
	for _, event := range auditLogger.events {
		if event.Status == "review" {
			foundReview = true
		}
		if event.Status == "success" {
			foundSuccess = true
		}
	}
	if !foundReview {
		t.Fatal("期望存在入库 review 审计事件")
	}
	if !foundSuccess {
		t.Fatal("期望存在入库 success 审计事件")
	}
}
