package engine

import (
	"context"
	"errors"
	"testing"

	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent"
	"flexirag-engine/internal/core/ports"
	"flexirag-engine/internal/core/review"
)

type fakeLLM struct {
	chatResp string
	embedVec [][]float32
}

func (f *fakeLLM) Chat(ctx context.Context, messages []core.Message) (string, error) {
	return f.chatResp, nil
}

func (f *fakeLLM) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(f.embedVec) > 0 {
		return f.embedVec, nil
	}
	return [][]float32{{0.1, 0.2}}, nil
}

type fakeVector struct {
	searchResp []core.SearchResult
}

func (f *fakeVector) Upsert(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	return nil
}

func (f *fakeVector) Search(ctx context.Context, agentID uint, vector []float32, topK int) ([]core.SearchResult, error) {
	return f.searchResp, nil
}

func (f *fakeVector) Delete(ctx context.Context, id string) error {
	return nil
}

type fakeReviewer struct {
	fn func(ctx context.Context, scope review.Scope, content string) (*review.Result, error)
}

func (f *fakeReviewer) Review(ctx context.Context, scope review.Scope, content string) (*review.Result, error) {
	return f.fn(ctx, scope, content)
}

var _ ports.LLMProvider = (*fakeLLM)(nil)
var _ ports.VectorStore = (*fakeVector)(nil)
var _ review.Reviewer = (*fakeReviewer)(nil)

func TestProcessQuery_InputRejectReturnsStructuredError(t *testing.T) {
	e := NewAgentEngine(
		&fakeLLM{chatResp: "unused"},
		&fakeVector{},
		WithReviewer(&fakeReviewer{
			fn: func(ctx context.Context, scope review.Scope, content string) (*review.Result, error) {
				if scope == review.ScopeInput {
					return &review.Result{
						Passed:  false,
						Action:  review.ActionReject,
						Message: "命中高风险规则，已拦截",
					}, nil
				}
				return &review.Result{Passed: true, Action: review.ActionAllow}, nil
			},
		}),
	)

	_, _, err := e.ProcessQuery(context.Background(), &agent.Agent{ID: 1, SystemPrompt: "test"}, "危险输入")
	if err == nil {
		t.Fatal("期望返回输入审核拒绝错误，实际为 nil")
	}

	var rejectErr *ReviewRejectedError
	if !errors.As(err, &rejectErr) {
		t.Fatalf("期望为 ReviewRejectedError，实际 %T", err)
	}
	if rejectErr.Scope != review.ScopeInput {
		t.Fatalf("期望 scope=input，实际 %s", rejectErr.Scope)
	}
}

func TestProcessQuery_OutputRejectReturnsStructuredError(t *testing.T) {
	e := NewAgentEngine(
		&fakeLLM{chatResp: "回答内容"},
		&fakeVector{searchResp: []core.SearchResult{
			{ID: "c1", Content: "知识片段", Score: 0.9},
		}},
		WithReviewer(&fakeReviewer{
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
		}),
	)

	_, _, err := e.ProcessQuery(context.Background(), &agent.Agent{ID: 1, SystemPrompt: "test"}, "正常问题")
	if err == nil {
		t.Fatal("期望返回输出审核拒绝错误，实际为 nil")
	}

	var rejectErr *ReviewRejectedError
	if !errors.As(err, &rejectErr) {
		t.Fatalf("期望为 ReviewRejectedError，实际 %T", err)
	}
	if rejectErr.Scope != review.ScopeOutput {
		t.Fatalf("期望 scope=output，实际 %s", rejectErr.Scope)
	}
}
