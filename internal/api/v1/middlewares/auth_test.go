package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"flexirag-engine/internal/core"

	"github.com/gin-gonic/gin"
)

type mockAuth struct {
	ok bool
}

func (m *mockAuth) ValidateToken(ctx context.Context, token string) (*core.Subject, error) {
	if m.ok && token == "ok" {
		return &core.Subject{ID: "admin"}, nil
	}
	return nil, errors.New("bad token")
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(&mockAuth{ok: true}))
	r.GET("/t", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("Authorization", "Bearer ok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/t", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际 %d", w2.Code)
	}
}
