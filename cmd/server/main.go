package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"flexirag-engine/internal/engine"
	"flexirag-engine/internal/model"
	"flexirag-engine/pkg/llm"
	"flexirag-engine/pkg/vector"

	"github.com/gin-gonic/gin"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请先设置环境变量 OPENAI_API_KEY")
	}
	// 在这里启动你的服务器，使用 apiKey 进行 OpenAI API 的调用
	llmProvider := llm.NewGLMClient(apiKey)
	vectorStore := vector.NewMockVectorStore()

	agentEngine := engine.NewAgentEngine(llmProvider, vectorStore)

	// 测试，灌入一些数据
	ctx := context.Background()
	mockAgent := setupMockData(ctx, llmProvider, vectorStore)

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// 聊天接口
	r.POST("/api/v1/chat", func(c *gin.Context) {
		// 定义前端传过来的请求体格式
		var req struct {
			Query string `json:"query" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，需要 query 字段"})
			return
		}

		// 调用核心引擎处理提问
		// 传入 request 的 Context，如果前端断开连接，能顺着链路取消大模型调用
		answer, err := agentEngine.ProcessQuery(c.Request.Context(), mockAgent, req.Query)
		if err != nil {
			log.Printf("处理失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 思考失败，请稍后再试"})
			return
		}

		// 返回大模型的回答
		c.JSON(http.StatusOK, gin.H{
			"answer": answer,
		})
	})

	// 6. 监听端口并启动服务
	fmt.Println("🚀 FlexiRAG Engine 启动成功！监听端口 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}

}

// setupMockData 仅用于 MVP 阶段的测试数据初始化
func setupMockData(ctx context.Context, llmProvider *llm.GLMClient, vectorStore *vector.MockVectorStore) *model.Agent {
	fmt.Println("正在初始化 Mock Agent 与向量知识库 (这会消耗一点点 Token)...")

	// 1. 创建一个教务小助手 Agent
	mockAgent := &model.Agent{
		ID:           1,
		Name:         "教务小助手",
		SystemPrompt: "你是 FlexiRAG 大学的教务助理。请严谨、礼貌地回答学生问题。如果问题不在你的上下文中，请说“抱歉，教务处目前没有相关通知”。",
	}

	// 2. 准备两条私有知识
	texts := []string{
		"FlexiRAG 大学 2026 年秋季四六级考试报名时间为 9 月 10 日至 9 月 20 日，报名费为 30 元，请在教务系统线上缴费。",
		"FlexiRAG 大学今年的暑假放假时间为 7 月 15 日，开学报到时间为 9 月 1 日。",
	}

	// 3. 将知识转化为向量
	vectors, err := llmProvider.Embed(ctx, texts)
	if err != nil {
		log.Fatalf("初始化数据 Embed 失败: %v", err)
	}

	// 4. 存入内存向量库 (注意：一定要传入 agent_id = 1)
	for i, text := range texts {
		id := fmt.Sprintf("knowledge_%d", i)
		metadata := map[string]interface{}{
			"content":  text,
			"agent_id": mockAgent.ID,
		}
		_ = vectorStore.Upsert(ctx, id, vectors[i], metadata)
	}

	fmt.Println("✅ 知识库初始化完成！")
	return mockAgent
}
