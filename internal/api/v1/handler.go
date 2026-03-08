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

type apiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, apiResponse{
		Code: http.StatusOK,
		Msg:  "success",
		Data: data,
	})
}

func respondError(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, apiResponse{
		Code: httpStatus,
		Msg:  msg,
		Data: nil,
	})
}

func NewHandler(agentEngine *engine.AgentEngine, chunkService *knowledge.ChunkService, agentRepo core.AgentRepository) *Handler {
	return &Handler{
		agentEngine:  agentEngine,
		chunkService: chunkService,
		agentRepo:    agentRepo,
	}
}

func (h *Handler) Ping(c *gin.Context) {
	respondSuccess(c, gin.H{"message": "pong"})
}

func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		Query   string `json:"query" binding:"required"`
		AgentID uint   `json:"agent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "参数错误，需要 query 和 agent_id 字段")
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
		return
	}
	if agent == nil {
		respondError(c, http.StatusNotFound, "Agent 不存在")
		return
	}

	answer, err := h.agentEngine.ProcessQuery(c.Request.Context(), agent, req.Query)
	if err != nil {
		log.Printf("处理失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "AI 思考失败，请稍后再试")
		return
	}

	respondSuccess(c, gin.H{"answer": answer})
}

func (h *Handler) IngestKnowledge(c *gin.Context) {
	var req struct {
		Text      string `json:"text" binding:"required"`
		AgentID   uint   `json:"agent_id" binding:"required"`
		ChunkSize int    `json:"chunk_size"`
		Overlap   int    `json:"overlap"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "参数错误，需要 text 和 agent_id 字段")
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
		return
	}
	if agent == nil {
		respondError(c, http.StatusNotFound, "Agent 不存在")
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
		respondError(c, http.StatusBadRequest, "overlap 必须小于 chunk_size")
		return
	}

	err = h.chunkService.IngestText(c.Request.Context(), req.AgentID, req.Text, chunkSize, overlap)
	if err != nil {
		log.Printf("知识入库失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "知识入库失败")
		return
	}

	respondSuccess(c, gin.H{
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
		respondError(c, http.StatusBadRequest, "参数错误，需要 name 和 system_prompt 字段")
		return
	}

	name := strings.TrimSpace(req.Name)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if name == "" || systemPrompt == "" {
		respondError(c, http.StatusBadRequest, "name 和 system_prompt 不能为空")
		return
	}

	agent := &agent_mgmt.Agent{
		Name:         name,
		SystemPrompt: systemPrompt,
	}

	if err := h.agentRepo.Create(c.Request.Context(), agent); err != nil {
		log.Printf("创建 Agent 失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "创建 Agent 失败")
		return
	}

	respondSuccess(c, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"system_prompt": agent.SystemPrompt,
	})
}

func (h *Handler) ListAgents(c *gin.Context) {
	agents, err := h.agentRepo.List(c.Request.Context())
	if err != nil {
		log.Printf("查询 Agent 花名册失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "查询 Agent 花名册失败")
		return
	}

	respondSuccess(c, gin.H{"agents": agents})
}

func (h *Handler) UpdateAgent(c *gin.Context) {
	idVal, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idVal == 0 {
		respondError(c, http.StatusBadRequest, "无效的 Agent ID")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		SystemPrompt *string `json:"system_prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "参数错误，JSON 格式不合法")
		return
	}

	var namePtr *string
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			respondError(c, http.StatusBadRequest, "name 不能为空")
			return
		}
		namePtr = &name
	}

	var promptPtr *string
	if req.SystemPrompt != nil {
		systemPrompt := strings.TrimSpace(*req.SystemPrompt)
		if systemPrompt == "" {
			respondError(c, http.StatusBadRequest, "system_prompt 不能为空")
			return
		}
		promptPtr = &systemPrompt
	}

	if namePtr == nil && promptPtr == nil {
		respondError(c, http.StatusBadRequest, "至少提供 name 或 system_prompt 其中一个字段")
		return
	}

	agent, err := h.agentRepo.Update(c.Request.Context(), uint(idVal), namePtr, promptPtr)
	if err != nil {
		log.Printf("更新 Agent 失败: %v\n", err)
		respondError(c, http.StatusInternalServerError, "更新 Agent 失败")
		return
	}
	if agent == nil {
		respondError(c, http.StatusNotFound, "Agent 不存在")
		return
	}

	respondSuccess(c, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"system_prompt": agent.SystemPrompt,
	})
}
