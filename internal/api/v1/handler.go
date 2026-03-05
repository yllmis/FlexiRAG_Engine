package v1

import (
	"log"
	"net/http"

	"flexirag-engine/internal/core/agent_mgmt"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	agentEngine  *engine.AgentEngine
	chunkService *knowledge.ChunkService
}

func NewHandler(agentEngine *engine.AgentEngine, chunkService *knowledge.ChunkService) *Handler {
	return &Handler{
		agentEngine:  agentEngine,
		chunkService: chunkService,
	}
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		Query   string `json:"query" binding:"required"`
		AgentID uint   `json:"agent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 query 字段"})
		return
	}

	agentID := req.AgentID
	if agentID == 0 {
		agentID = 1
	}

	currentAgent := &agent_mgmt.Agent{
		ID:           agentID,
		Name:         "智能助手",
		SystemPrompt: "你是一个专业的AI助手。请严格根据检索到的上下文信息回答问题。如果上下文中没有提及，请直接回答“抱歉，我的知识库中没有相关信息”。",
	}

	answer, err := h.agentEngine.ProcessQuery(c.Request.Context(), currentAgent, req.Query)
	if err != nil {
		log.Printf("处理失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 思考失败，请稍后再试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"answer": answer})
}

func (h *Handler) IngestKnowledge(c *gin.Context) {
	var req struct {
		Text      string `json:"text" binding:"required"`
		AgentID   uint   `json:"agent_id"`
		ChunkSize int    `json:"chunk_size"`
		Overlap   int    `json:"overlap"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 text 字段"})
		return
	}

	agentID := req.AgentID
	if agentID == 0 {
		agentID = 1
	}

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 300
	}

	overlap := req.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "overlap 必须小于 chunk_size"})
		return
	}

	err := h.chunkService.IngestText(c.Request.Context(), agentID, req.Text, chunkSize, overlap)
	if err != nil {
		log.Printf("知识入库失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "知识入库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "知识入库成功，已持久化到 PostgreSQL",
		"agent_id":   agentID,
		"chunk_size": chunkSize,
		"overlap":    overlap,
	})
}
