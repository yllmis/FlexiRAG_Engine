package v1

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"flexirag-engine/internal/api/v1/middlewares"
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
	auditLogger  core.AuditLogger
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

func NewHandler(agentEngine *engine.AgentEngine, chunkService *knowledge.ChunkService, agentRepo core.AgentRepository, auditLogger ...core.AuditLogger) *Handler {
	h := &Handler{
		agentEngine:  agentEngine,
		chunkService: chunkService,
		agentRepo:    agentRepo,
	}
	if len(auditLogger) > 0 {
		h.auditLogger = auditLogger[0]
	}
	return h
}

func (h *Handler) audit(c *gin.Context, eventType, resourceType, resourceID, status, msg string) {
	if h.auditLogger == nil {
		return
	}
	subjectID := "anonymous"
	if v, ok := c.Get(middlewares.ContextSubjectIDKey); ok {
		subjectID = strings.TrimSpace(v.(string))
	}
	requestID := ""
	if v, ok := c.Get(middlewares.ContextRequestIDKey); ok {
		requestID = strings.TrimSpace(v.(string))
	}
	h.auditLogger.Log(core.AuditEvent{
		EventType:    eventType,
		SubjectID:    subjectID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		RequestID:    requestID,
		Status:       status,
		Message:      msg,
	})
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
		h.audit(c, "chat", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "参数错误")
		respondError(c, http.StatusBadRequest, "参数错误，需要 query 和 agent_id 字段")
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		h.audit(c, "chat", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "查询 Agent 失败")
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
		return
	}
	if agent == nil {
		h.audit(c, "chat", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "Agent 不存在")
		respondError(c, http.StatusNotFound, "Agent 不存在")
		return
	}

	answer, err := h.agentEngine.ProcessQuery(c.Request.Context(), agent, req.Query)
	if err != nil {
		log.Printf("处理失败: %v\n", err)
		h.audit(c, "chat", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "AI 思考失败")
		respondError(c, http.StatusInternalServerError, "AI 思考失败，请稍后再试")
		return
	}

	h.audit(c, "chat", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "success", "问答成功")
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
		h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "参数错误")
		respondError(c, http.StatusBadRequest, "参数错误，需要 text 和 agent_id 字段")
		return
	}

	agent, err := h.agentRepo.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		log.Printf("查询 Agent 失败: %v\n", err)
		h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "查询 Agent 失败")
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
		return
	}
	if agent == nil {
		h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "Agent 不存在")
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
		h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "overlap 参数不合法")
		respondError(c, http.StatusBadRequest, "overlap 必须小于 chunk_size")
		return
	}

	err = h.chunkService.IngestText(c.Request.Context(), req.AgentID, req.Text, chunkSize, overlap)
	if err != nil {
		log.Printf("知识入库失败: %v\n", err)
		h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "failed", "知识入库失败")
		respondError(c, http.StatusInternalServerError, "知识入库失败")
		return
	}

	h.audit(c, "knowledge_ingest", "agent", strconv.FormatUint(uint64(req.AgentID), 10), "success", "知识入库成功")
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
		h.audit(c, "agent_create", "agent", "", "failed", "参数错误")
		respondError(c, http.StatusBadRequest, "参数错误，需要 name 和 system_prompt 字段")
		return
	}

	name := strings.TrimSpace(req.Name)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if name == "" || systemPrompt == "" {
		h.audit(c, "agent_create", "agent", "", "failed", "name 或 system_prompt 为空")
		respondError(c, http.StatusBadRequest, "name 和 system_prompt 不能为空")
		return
	}

	agent := &agent_mgmt.Agent{
		Name:         name,
		SystemPrompt: systemPrompt,
	}

	if err := h.agentRepo.Create(c.Request.Context(), agent); err != nil {
		log.Printf("创建 Agent 失败: %v\n", err)
		h.audit(c, "agent_create", "agent", "", "failed", "创建 Agent 失败")
		respondError(c, http.StatusInternalServerError, "创建 Agent 失败")
		return
	}

	h.audit(c, "agent_create", "agent", strconv.FormatUint(uint64(agent.ID), 10), "success", "创建 Agent 成功")
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
		h.audit(c, "agent_update", "agent", c.Param("id"), "failed", "Agent ID 非法")
		respondError(c, http.StatusBadRequest, "无效的 Agent ID")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		SystemPrompt *string `json:"system_prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "JSON 非法")
		respondError(c, http.StatusBadRequest, "参数错误，JSON 格式不合法")
		return
	}

	var namePtr *string
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "name 为空")
			respondError(c, http.StatusBadRequest, "name 不能为空")
			return
		}
		namePtr = &name
	}

	var promptPtr *string
	if req.SystemPrompt != nil {
		systemPrompt := strings.TrimSpace(*req.SystemPrompt)
		if systemPrompt == "" {
			h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "system_prompt 为空")
			respondError(c, http.StatusBadRequest, "system_prompt 不能为空")
			return
		}
		promptPtr = &systemPrompt
	}

	if namePtr == nil && promptPtr == nil {
		h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "更新字段为空")
		respondError(c, http.StatusBadRequest, "至少提供 name 或 system_prompt 其中一个字段")
		return
	}

	agent, err := h.agentRepo.Update(c.Request.Context(), uint(idVal), namePtr, promptPtr)
	if err != nil {
		log.Printf("更新 Agent 失败: %v\n", err)
		h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "更新 Agent 失败")
		respondError(c, http.StatusInternalServerError, "更新 Agent 失败")
		return
	}
	if agent == nil {
		h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "failed", "Agent 不存在")
		respondError(c, http.StatusNotFound, "Agent 不存在")
		return
	}

	h.audit(c, "agent_update", "agent", strconv.FormatUint(idVal, 10), "success", "更新 Agent 成功")
	respondSuccess(c, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"system_prompt": agent.SystemPrompt,
	})
}

func (h *Handler) DeleteAgent(c *gin.Context) {
	idVal, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idVal == 0 {
		h.audit(c, "agent_delete", "agent", c.Param("id"), "failed", "Agent ID 非法")
		respondError(c, http.StatusBadRequest, "无效的 Agent ID")
		return
	}

	ok, err := h.agentRepo.Delete(c.Request.Context(), uint(idVal))
	if err != nil {
		log.Printf("删除 Agent 失败: %v\n", err)
		h.audit(c, "agent_delete", "agent", strconv.FormatUint(idVal, 10), "failed", "删除 Agent 失败")
		respondError(c, http.StatusInternalServerError, "删除 Agent 失败")
		return
	}
	if !ok {
		h.audit(c, "agent_delete", "agent", strconv.FormatUint(idVal, 10), "failed", "Agent 不存在")
		respondError(c, http.StatusNotFound, "Agent 不存在")
		return
	}

	h.audit(c, "agent_delete", "agent", strconv.FormatUint(idVal, 10), "success", "删除 Agent 成功")
	respondSuccess(c, gin.H{
		"agent_id": idVal,
		"deleted":  true,
	})
}
