package v1

import "github.com/gin-gonic/gin"

// SetupRouter 注册所有的 V1 API 路由
func SetupRouter(r *gin.Engine, h *Handler) {
	r.GET("/ping", h.Ping)

	apiV1 := r.Group("/api/v1")
	{
		apiV1.POST("/agents", h.CreateAgent)
		apiV1.GET("/agents", h.ListAgents)
		apiV1.PATCH("/agents/:id/system-prompt", h.UpdateAgentSystemPrompt)
		apiV1.POST("/chat", h.Chat)
		apiV1.POST("/knowledge/ingest", h.IngestKnowledge)
	}
}

func RegisterRoutes(r *gin.Engine, h *Handler) {
	SetupRouter(r, h)
}
