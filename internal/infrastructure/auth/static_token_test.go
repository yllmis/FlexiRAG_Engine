package auth

import (
	"context"
	"testing"
)

func TestStaticTokenAuth_ValidateToken(t *testing.T) {
	a := NewStaticTokenAuth("token-123")

	subject, err := a.ValidateToken(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("期望校验成功，实际错误: %v", err)
	}
	if subject == nil || subject.ID != "admin" {
		t.Fatalf("期望 subject=admin，实际=%v", subject)
	}

	if _, err := a.ValidateToken(context.Background(), "bad"); err == nil {
		t.Fatal("期望错误 token 校验失败")
	}
}
