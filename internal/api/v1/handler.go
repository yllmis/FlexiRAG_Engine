package v1

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent_mgmt"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	agentEngine  *engine.AgentEngine
	chunkService *knowledge.ChunkService
	agentRepo    core.AgentRepository
}

func NewHandler(agentEngine *engine.AgentEngine, chunkService *knowledge.ChunkService, agentRepo core.AgentRepository) *Handler {
	return &Handler{
		agentEngine:  agentEngine,
		chunkService: chunkService,
		agentRepo:    agentRepo,
	}
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		Query   string `json:"query" binding:"required"`
		AgentID uint   `json:"agent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 query 和 agent_id 字段"})
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询 Agent 失败"})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent 不存在"})
		return
	}

	answer, err := h.agentEngine.ProcessQuery(c.Request.Context(), agent, req.Query)
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
		AgentID   uint   `json:"agent_id" binding:"required"`
		ChunkSize int    `json:"chunk_size"`
		Overlap   int    `json:"overlap"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 text 和 agent_id 字段"})
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询 Agent 失败"})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent 不存在"})
		return
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

	err = h.chunkService.IngestText(c.Request.Context(), req.AgentID, req.Text, chunkSize, overlap)
	if err != nil {
		log.Printf("知识入库失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "知识入库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "知识入库成功，已持久化到 PostgreSQL",
		"agent_id":   req.AgentID,
		"chunk_size": chunkSize,
		"overlap":    overlap,
	})
}

func (h *Handler) CreateAgent(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		SystemPrompt string `json:"system_prompt" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 name 和 system_prompt 字段"})
		return
	}

	name := strings.TrimSpace(req.Name)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if name == "" || systemPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 和 system_prompt 不能为空"})
		return
	}

	agent := &agent_mgmt.Agent{
		Name:         name,
		SystemPrompt: systemPrompt,
	}

	if err := h.agentRepo.Create(c.Request.Context(), agent); err != nil {
		log.Printf("创建 Agent 失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Agent 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"system_prompt": agent.SystemPrompt,
	})
}

func (h *Handler) ListAgents(c *gin.Context) {
	agents, err := h.agentRepo.List(c.Request.Context())
	if err != nil {
		log.Printf("查询 Agent 花名册失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询 Agent 花名册失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *Handler) UpdateAgentSystemPrompt(c *gin.Context) {
	idVal, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idVal == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Agent ID"})
		return
	}

	var req struct {
		SystemPrompt string `json:"system_prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 system_prompt 字段"})
		return
	}

	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if systemPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system_prompt 不能为空"})
		return
	}

	agent, err := h.agentRepo.UpdateSystemPrompt(c.Request.Context(), uint(idVal), systemPrompt)
	if err != nil {
		log.Printf("更新 Agent 系统提示词失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Agent 系统提示词失败"})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent 不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"system_prompt": agent.SystemPrompt,
	})
}
