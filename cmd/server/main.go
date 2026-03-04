package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"flexirag-engine/internal/core/agent_mgmt"
	"flexirag-engine/internal/core/knowledge"
	"flexirag-engine/internal/engine"
	"flexirag-engine/internal/infrastructure/llm"
	"flexirag-engine/internal/infrastructure/vector"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请先设置环境变量 OPENAI_API_KEY")
	}

	llmProvider := llm.NewGLMClient(apiKey)
	dsn := "host=localhost user=root password=12345 dbname=flexirag_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接 PostgreSQL 失败: ", err)
	}

	vectorStore, err := vector.NewPGVectorStore(db)
	if err != nil {
		log.Fatal("初始化 PG 向量库失败: ", err)
	}

	agentEngine := engine.NewAgentEngine(llmProvider, vectorStore)
	chunkService := knowledge.NewChunkService(llmProvider, vectorStore)

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/api/v1/chat", func(c *gin.Context) {
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

		answer, err := agentEngine.ProcessQuery(c.Request.Context(), currentAgent, req.Query)
		if err != nil {
			log.Printf("处理失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 思考失败，请稍后再试"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"answer": answer})
	})

	r.POST("/api/v1/knowledge/ingest", func(c *gin.Context) {
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

		err := chunkService.IngestText(c.Request.Context(), agentID, req.Text, chunkSize, overlap)
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
	})

	fmt.Println("🚀 FlexiRAG Engine 启动成功！监听端口 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}

func setupMockData(ctx context.Context, llmProvider *llm.GLMClient, vectorStore *vector.MockVectorStore) *agent_mgmt.Agent {
	fmt.Println("正在启动 ChunkService 自动切片并录入长篇知识库...")

	mockAgent := &agent_mgmt.Agent{
		ID:           1,
		Name:         "教务小助手",
		SystemPrompt: "你是 FlexiRAG 大学的教务助理。请严谨、礼貌地依据上下文回答问题。如果资料里没有，请说不知道。",
	}

	longDocument := `FlexiRAG 大学 2026 年新生入学指南与教务通知。
第一章：报到与住宿。今年的暑假放假时间为 7 月 15 日。新生开学报到时间统一安排在 9 月 1 日，请务必携带录取通知书原件。新生宿舍分配将在 8 月 25 日通过教务系统官网公布，请同学们自行登录查询。
第二章：关于英语四六级考试。为了保证考试资源的合理分配，大一新生第一学期不允许报考英语四级。2026 年秋季四六级考试的报名时间为 9 月 10 日至 9 月 20 日，报名费为 30 元。请注意，所有的缴费均须在教务系统线上完成，学校不会安排任何老师私下收取微信转账。
第三章：校园生活。学校目前共有三个食堂，其中第二食堂的麻辣烫最受学生欢迎，营业时间为早上 7 点到晚上 10 点。`

	chunkService := knowledge.NewChunkService(llmProvider, vectorStore)
	err := chunkService.IngestText(ctx, mockAgent.ID, longDocument, 100, 20)
	if err != nil {
		log.Fatalf("知识库长文录入失败: %v", err)
	}

	fmt.Println("✅ 长文切片与知识库录入完成！你可以开始提问了。")
	return mockAgent
}
