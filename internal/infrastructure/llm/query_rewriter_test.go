package llm

import (
	"context"
	"errors"
	"testing"

	"flexirag-engine/internal/core"
)

type mockLLMForRewrite struct {
	chatFn func(ctx context.Context, messages []core.Message) (string, error)
}

func (m *mockLLMForRewrite) Chat(ctx context.Context, messages []core.Message) (string, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, messages)
	}
	return "", nil
}

func (m *mockLLMForRewrite) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

func TestRewrite_Success(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "2026年英语四六级考试报名时间\n四六级考试报名截止日期\n四六级报名缴费方式", nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "那个报名时间呢？")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d: %v", len(queries), queries)
	}
	if queries[0] != "2026年英语四六级考试报名时间" {
		t.Fatalf("unexpected first query: %s", queries[0])
	}
}

func TestRewrite_MultiQuery(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "1. 四六级考试报名时间\n2. 四六级考试缴费方式\n3. 四六级线上缴费要求", nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "四六级报名时间是什么时候，怎么缴费？")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d: %v", len(queries), queries)
	}
	// 编号应该被去除
	if queries[0] != "四六级考试报名时间" {
		t.Fatalf("unexpected first query: %s", queries[0])
	}
}

func TestRewrite_FallbackOnError(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "", errors.New("LLM 超时")
		},
	}

	// fallback=true（默认）
	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "测试问题")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "测试问题" {
		t.Fatalf("expected fallback to original, got %v", queries)
	}

	// fallback=false
	r2 := NewLLMQueryRewriter(mock, WithFallback(false))
	_, err = r2.Rewrite(context.Background(), "测试问题")
	if err == nil {
		t.Fatal("expected error when fallback disabled")
	}
}

func TestRewrite_EmptyQuery(t *testing.T) {
	mock := &mockLLMForRewrite{}
	r := NewLLMQueryRewriter(mock)

	queries, err := r.Rewrite(context.Background(), "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "" {
		t.Fatalf("expected [\"\"], got %v", queries)
	}
}

func TestRewrite_EmptyLLMResponse(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "   \n  \n  ", nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "原始问题")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "原始问题" {
		t.Fatalf("expected fallback to original, got %v", queries)
	}
}

func TestRewrite_WithMaxQueries(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "q1\nq2\nq3\nq4\nq5", nil
		},
	}

	r := NewLLMQueryRewriter(mock, WithMaxQueries(5))
	queries, err := r.Rewrite(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 5 {
		t.Fatalf("expected 5 queries, got %d", len(queries))
	}
}

func TestStripCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"json block", "```json\n{\"k\":\"v\"}\n```", `{"k":"v"}`},
		{"plain block", "```\n{\"k\":\"v\"}\n```", `{"k":"v"}`},
		{"no block", `{"k":"v"}`, `{"k":"v"}`},
		{"with spaces", "  ```json\n{\"k\":\"v\"}\n```  ", `{"k":"v"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("stripCodeBlock(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRewrite_JSONInCodeBlock(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "```json\n{\"sub_queries\": [\"q1\", \"q2\"], \"reasoning\": \"拆分\"}\n```", nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d: %v", len(queries), queries)
	}
	if queries[0] != "q1" || queries[1] != "q2" {
		t.Fatalf("unexpected queries: %v", queries)
	}
}

func TestRewrite_JSONResponse(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return `{"sub_queries": ["四六级考试报名时间", "四六级考试缴费方式"], "reasoning": "拆分为两个独立意图"}`, nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "四六级报名时间是什么时候，怎么缴费？")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d: %v", len(queries), queries)
	}
	if queries[0] != "四六级考试报名时间" || queries[1] != "四六级考试缴费方式" {
		t.Fatalf("unexpected queries: %v", queries)
	}
}

func TestRewrite_JSONWithDuplicates(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return `{"sub_queries": ["q1", "q2", "q1"]}`, nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries after dedup, got %d: %v", len(queries), queries)
	}
}

func TestRewrite_MalformedJSONFallsBackToText(t *testing.T) {
	mock := &mockLLMForRewrite{
		chatFn: func(_ context.Context, _ []core.Message) (string, error) {
			return "这不是JSON\n而是普通文本", nil
		},
	}

	r := NewLLMQueryRewriter(mock)
	queries, err := r.Rewrite(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries from text fallback, got %d: %v", len(queries), queries)
	}
}

func TestParseQueries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"numbered", "1. q1\n2. q2\n3. q3", 3},
		{"dashes", "- q1\n- q2", 2},
		{"mixed", "1、q1\n2）q2\n- q3", 3},
		{"duplicates", "q1\nq2\nq1", 2},
		{"empty lines", "\nq1\n\nq2\n\n", 2},
		{"chinese numbering", "一、q1\n二、q2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseQueries(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}
