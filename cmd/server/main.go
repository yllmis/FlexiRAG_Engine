package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	apiV1 "flexirag-engine/internal/api/v1"
	"flexirag-engine/internal/config"
	"flexirag-engine/internal/core/agent"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/engine"
	"flexirag-engine/internal/infrastructure/audit"
	"flexirag-engine/internal/infrastructure/auth"
	"flexirag-engine/internal/infrastructure/database"
	"flexirag-engine/internal/infrastructure/llm"
	"flexirag-engine/internal/infrastructure/ratelimit"
	"flexirag-engine/internal/infrastructure/repository"
	"flexirag-engine/internal/infrastructure/vector"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
)

func main() {
	cfg, err := config.Load(os.Getenv("APP_CONFIG_PATH"))
	if err != nil {
		log.Fatal("加载配置失败: ", err)
	}

	llmProvider := llm.NewGLMClientWithConfig(
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.ChatModel,
		openai.EmbeddingModel(cfg.LLM.EmbedModel),
	)

	db, err := database.NewPostgresDB(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		TimeZone: cfg.Database.TimeZone,
	})
	if err != nil {
		log.Fatal("连接 PostgreSQL 失败: ", err)
	}

	vectorStore, err := vector.NewPGVectorStore(db)
	if err != nil {
		log.Fatal("初始化 PG 向量库失败: ", err)
	}
	agentRepo, err := repository.NewPGAgentRepo(db)
	if err != nil {
		log.Fatal("初始化 Agent 仓储失败: ", err)
	}
	auditRepo, err := repository.NewPGAuditRepo(db)
	if err != nil {
		log.Fatal("初始化审计仓储失败: ", err)
	}

	rewriter := llm.NewLLMQueryRewriter(llmProvider)

	agentEngine := engine.NewAgentEngine(llmProvider, vectorStore, engine.WithQueryRewriter(rewriter))
	chunkService := knowledge.NewChunkService(llmProvider, vectorStore)
	agentSvc := agent.NewAgentService(agentRepo)
	auditLogger := audit.NewAsyncWriter(auditRepo, cfg.Security.AuditQueueSize)
	authService := auth.NewStaticTokenAuth(cfg.Security.AdminToken)
	rateLimiter := ratelimit.NewInMemoryRateLimiter(cfg.Security.RateLimitPerMinute)

	r := gin.Default()
	handler := apiV1.NewHandler(agentEngine, chunkService, agentSvc, auditLogger)
	apiV1.RegisterRoutes(r, handler, authService, rateLimiter)

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	fmt.Printf("🚀 FlexiRAG Engine 启动成功！监听端口 %s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}
