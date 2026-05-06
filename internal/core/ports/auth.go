package ports

import (
	"context"
	"flexirag-engine/internal/core"
)

// AuthService 定义鉴权服务标准
type AuthService interface {
	ValidateToken(ctx context.Context, token string) (*core.Subject, error)
}