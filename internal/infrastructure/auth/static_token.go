package auth

import (
	"context"
	"errors"
	"strings"

	"flexirag-engine/internal/core"
)

var ErrInvalidToken = errors.New("invalid token")

var _ core.AuthService = (*StaticTokenAuth)(nil)

type StaticTokenAuth struct {
	adminToken string
}

func NewStaticTokenAuth(adminToken string) *StaticTokenAuth {
	return &StaticTokenAuth{adminToken: strings.TrimSpace(adminToken)}
}

func (s *StaticTokenAuth) ValidateToken(_ context.Context, token string) (*core.Subject, error) {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(token) != s.adminToken {
		return nil, ErrInvalidToken
	}
	return &core.Subject{ID: "admin"}, nil
}
