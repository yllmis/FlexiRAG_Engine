package v1

import (
	"flexirag-engine/internal/api/v1/middlewares"
	"flexirag-engine/internal/core"

	"github.com/gin-gonic/gin"
)

// SetupRouter 注册所有的 V1 API 路由
func SetupRouter(r *gin.Engine, h *Handler, authService core.AuthService, limiter core.RateLimiter) {
	r.Use(middlewares.RequestID())
	r.GET("/ping", h.Ping)

	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/agents", h.ListAgents)

		protected := apiV1.Group("")
		protected.Use(middlewares.Auth(authService), middlewares.RateLimit(limiter))
		protected.POST("/agents", h.CreateAgent)
		protected.PUT("/agents/:id", h.UpdateAgent)
		protected.POST("/chat", h.Chat)
		protected.POST("/knowledge/ingest", h.IngestKnowledge)
	}
}

func RegisterRoutes(r *gin.Engine, h *Handler, authService core.AuthService, limiter core.RateLimiter) {
	SetupRouter(r, h, authService, limiter)
}
