package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	apiV1 "flexirag-engine/internal/api/v1"
	"flexirag-engine/internal/config"
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

	agentEngine := engine.NewAgentEngine(llmProvider, vectorStore)
	chunkService := knowledge.NewChunkService(llmProvider, vectorStore)
	auditLogger := audit.NewAsyncWriter(auditRepo, cfg.Security.AuditQueueSize)
	authService := auth.NewStaticTokenAuth(cfg.Security.AdminToken)
	rateLimiter := ratelimit.NewInMemoryRateLimiter(cfg.Security.RateLimitPerMinute)

	r := gin.Default()
	handler := apiV1.NewHandler(agentEngine, chunkService, agentRepo, auditLogger)
	apiV1.RegisterRoutes(r, handler, authService, rateLimiter)

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	fmt.Printf("🚀 FlexiRAG Engine 启动成功！监听端口 %s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}

// func setupMockData(ctx context.Context, llmProvider *llm.GLMClient, vectorStore *vector.MockVectorStore) *agent_mgmt.Agent {
// 	fmt.Println("正在启动 ChunkService 自动切片并录入长篇知识库...")

// 	mockAgent := &agent_mgmt.Agent{
// 		ID:           1,
// 		Name:         "教务小助手",
// 		SystemPrompt: "你是 FlexiRAG 大学的教务助理。请严谨、礼貌地依据上下文回答问题。如果资料里没有，请说不知道。",
// 	}

// 	longDocument := `FlexiRAG 大学 2026 年新生入学指南与教务通知。
// 第一章：报到与住宿。今年的暑假放假时间为 7 月 15 日。新生开学报到时间统一安排在 9 月 1 日，请务必携带录取通知书原件。新生宿舍分配将在 8 月 25 日通过教务系统官网公布，请同学们自行登录查询。
// 第二章：关于英语四六级考试。为了保证考试资源的合理分配，大一新生第一学期不允许报考英语四级。2026 年秋季四六级考试的报名时间为 9 月 10 日至 9 月 20 日，报名费为 30 元。请注意，所有的缴费均须在教务系统线上完成，学校不会安排任何老师私下收取微信转账。
// 第三章：校园生活。学校目前共有三个食堂，其中第二食堂的麻辣烫最受学生欢迎，营业时间为早上 7 点到晚上 10 点。`

// 	chunkService := knowledge.NewChunkService(llmProvider, vectorStore)
// 	err := chunkService.IngestText(ctx, mockAgent.ID, longDocument, 100, 20)
// 	if err != nil {
// 		log.Fatalf("知识库长文录入失败: %v", err)
// 	}

// 	fmt.Println("✅ 长文切片与知识库录入完成！你可以开始提问了。")
// 	return mockAgent
// }
