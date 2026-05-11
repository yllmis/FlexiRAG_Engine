package v1

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"flexirag-engine/internal/api/v1/middlewares"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/core/agent"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/core/ports"
	"flexirag-engine/internal/core/review"
	"flexirag-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	agentEngine  *engine.AgentEngine
	chunkService *knowledge.ChunkService
	agentSvc     agent.AgentService
	auditLogger  ports.AuditLogger
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

func NewHandler(agentEngine *engine.AgentEngine, chunkService *knowledge.ChunkService, agentSvc agent.AgentService, auditLogger ...ports.AuditLogger) *Handler {
	h := &Handler{
		agentEngine:  agentEngine,
		chunkService: chunkService,
		agentSvc:     agentSvc,
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
		h.audit(c, "chat", "agent", "", "failed", "参数错误")
		respondError(c, http.StatusBadRequest, "参数错误，需要 query 和 agent_id 字段")
		return
	}

	agt, err := h.agentSvc.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		agentIDStr := strconv.FormatUint(uint64(req.AgentID), 10)
		if errors.Is(err, agent.ErrAgentNotFound) {
			h.audit(c, "chat", "agent", agentIDStr, "failed", "Agent 不存在")
			respondError(c, http.StatusNotFound, "Agent 不存在")
			return
		}
		log.Printf("查询 Agent 失败: %v\n", err)
		h.audit(c, "chat", "agent", agentIDStr, "failed", "查询 Agent 失败")
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
		return
	}

	answer, outcome, err := h.agentEngine.ProcessQuery(c.Request.Context(), agt, req.Query)
	if err != nil {
		agentIDStr := strconv.FormatUint(uint64(req.AgentID), 10)
		var reviewErr *engine.ReviewRejectedError
		if errors.As(err, &reviewErr) {
			h.audit(c, "chat", "agent", agentIDStr, "rejected", err.Error())
			msg := "输入内容未通过审核"
			if reviewErr.Scope == review.ScopeOutput {
				msg = "输出内容未通过审核"
			}
			respondError(c, http.StatusForbidden, msg)
			return
		}
		log.Printf("处理失败: %v\n", err)
		h.audit(c, "chat", "agent", agentIDStr, "failed", "AI 思考失败")
		respondError(c, http.StatusInternalServerError, "AI 思考失败，请稍后再试")
		return
	}

	agentIDStr := strconv.FormatUint(uint64(req.AgentID), 10)

	// 处理审核结果（输入 warn/review 或输出 warn/review）
	if outcome != nil {
		switch outcome.Action {
		case review.ActionReview:
			h.audit(c, "chat", "agent", agentIDStr, "review", "命中审核规则，需人工复核")
		case review.ActionWarn:
			h.audit(c, "chat", "agent", agentIDStr, "warn", "命中审核规则，已记录警告")
		}
	}

	h.audit(c, "chat", "agent", agentIDStr, "success", "问答成功")
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
		h.audit(c, "knowledge_ingest", "agent", "", "failed", "参数错误")
		respondError(c, http.StatusBadRequest, "参数错误，需要 text 和 agent_id 字段")
		return
	}

	agentIDStr := strconv.FormatUint(uint64(req.AgentID), 10)
	agt, err := h.agentSvc.GetByID(c.Request.Context(), req.AgentID)
	if err != nil {
		if errors.Is(err, agent.ErrAgentNotFound) {
			h.audit(c, "knowledge_ingest", "agent", agentIDStr, "failed", "Agent 不存在")
			respondError(c, http.StatusNotFound, "Agent 不存在")
			return
		}
		log.Printf("查询 Agent 失败: %v\n", err)
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "failed", "查询 Agent 失败")
		respondError(c, http.StatusInternalServerError, "查询 Agent 失败")
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
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "failed", "overlap 参数不合法")
		respondError(c, http.StatusBadRequest, "overlap 必须小于 chunk_size")
		return
	}

	// 入库审核（可选，异常时 fail-closed）
	ingestResult, err := h.agentEngine.ReviewIngest(c.Request.Context(), req.Text)
	if err != nil {
		log.Printf("入库审核异常: %v\n", err)
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "failed", "入库审核异常")
		respondError(c, http.StatusInternalServerError, "入库审核异常，请稍后再试")
		return
	}
	if !ingestResult.Passed {
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "rejected", "入库内容未通过审核")
		respondError(c, http.StatusForbidden, "入库内容未通过审核")
		return
	}
	switch ingestResult.Action {
	case review.ActionReview:
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "review", "入库内容命中审核规则，需人工复核")
	case review.ActionWarn:
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "warn", "入库内容命中审核规则，已记录警告")
	}

	err = h.chunkService.IngestText(c.Request.Context(), agt.ID, req.Text, chunkSize, overlap)
	if err != nil {
		log.Printf("知识入库失败: %v\n", err)
		h.audit(c, "knowledge_ingest", "agent", agentIDStr, "failed", "知识入库失败")
		respondError(c, http.StatusInternalServerError, "知识入库失败")
		return
	}

	h.audit(c, "knowledge_ingest", "agent", agentIDStr, "success", "知识入库成功")
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

	agt, err := h.agentSvc.Create(c.Request.Context(), req.Name, req.SystemPrompt)
	if err != nil {
		if errors.Is(err, agent.ErrInvalidInput) {
			h.audit(c, "agent_create", "agent", "", "failed", "name 或 system_prompt 为空")
			respondError(c, http.StatusBadRequest, "name 和 system_prompt 不能为空")
			return
		}
		log.Printf("创建 Agent 失败: %v\n", err)
		h.audit(c, "agent_create", "agent", "", "failed", "创建 Agent 失败")
		respondError(c, http.StatusInternalServerError, "创建 Agent 失败")
		return
	}

	h.audit(c, "agent_create", "agent", strconv.FormatUint(uint64(agt.ID), 10), "success", "创建 Agent 成功")
	respondSuccess(c, gin.H{
		"agent_id":      agt.ID,
		"name":          agt.Name,
		"system_prompt": agt.SystemPrompt,
	})
}

func (h *Handler) ListAgents(c *gin.Context) {
	agents, err := h.agentSvc.List(c.Request.Context())
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

	agt, err := h.agentSvc.Update(c.Request.Context(), uint(idVal), req.Name, req.SystemPrompt)
	if err != nil {
		idStr := strconv.FormatUint(idVal, 10)
		if errors.Is(err, agent.ErrInvalidInput) {
			h.audit(c, "agent_update", "agent", idStr, "failed", "参数校验失败")
			respondError(c, http.StatusBadRequest, "参数错误：name 或 system_prompt 不能为空，且至少提供一个字段")
			return
		}
		if errors.Is(err, agent.ErrAgentNotFound) {
			h.audit(c, "agent_update", "agent", idStr, "failed", "Agent 不存在")
			respondError(c, http.StatusNotFound, "Agent 不存在")
			return
		}
		log.Printf("更新 Agent 失败: %v\n", err)
		h.audit(c, "agent_update", "agent", idStr, "failed", "更新 Agent 失败")
		respondError(c, http.StatusInternalServerError, "更新 Agent 失败")
		return
	}

	h.audit(c, "agent_update", "agent", strconv.FormatUint(uint64(agt.ID), 10), "success", "更新 Agent 成功")
	respondSuccess(c, gin.H{
		"agent_id":      agt.ID,
		"name":          agt.Name,
		"system_prompt": agt.SystemPrompt,
	})
}

func (h *Handler) DeleteAgent(c *gin.Context) {
	idVal, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idVal == 0 {
		h.audit(c, "agent_delete", "agent", c.Param("id"), "failed", "Agent ID 非法")
		respondError(c, http.StatusBadRequest, "无效的 Agent ID")
		return
	}

	err = h.agentSvc.Delete(c.Request.Context(), uint(idVal))
	if err != nil {
		idStr := strconv.FormatUint(idVal, 10)
		if errors.Is(err, agent.ErrAgentNotFound) {
			h.audit(c, "agent_delete", "agent", idStr, "failed", "Agent 不存在")
			respondError(c, http.StatusNotFound, "Agent 不存在")
			return
		}
		log.Printf("删除 Agent 失败: %v\n", err)
		h.audit(c, "agent_delete", "agent", idStr, "failed", "删除 Agent 失败")
		respondError(c, http.StatusInternalServerError, "删除 Agent 失败")
		return
	}

	h.audit(c, "agent_delete", "agent", strconv.FormatUint(idVal, 10), "success", "删除 Agent 成功")
	respondSuccess(c, gin.H{
		"agent_id": idVal,
		"deleted":  true,
	})
}
