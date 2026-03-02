package knowledge

import (
	"context"
	"flexirag-engine/internal/core"
	"flexirag-engine/internal/pkg/textsplit"
	"fmt"
	"time"
)

type ChunkService struct {
	llm    core.LLMProvider
	vector core.VectorStore
}

func NewChunkService(llm core.LLMProvider, vector core.VectorStore) *ChunkService {
	return &ChunkService{
		llm:    llm,
		vector: vector,
	}
}

func (s *ChunkService) IngestText(ctx context.Context, agentID uint, text string, chunkSize, overlop int) error {
	splitter := textsplit.NewTextSplitter(chunkSize, overlop, "")
	chunks := splitter.Split(text)

	if len(chunks) == 0 {
		return fmt.Errorf("没有提取到有效文本")
	}

	// 批量向量化
	vectors, err := s.llm.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}

	if len(vectors) != len(chunks) {
		return fmt.Errorf("向量化结果数量与文本块数量不匹配")
	}

	// 切片与向量一一绑定，并存储到向量数据库
	for i, chunkContent := range chunks {
		chunkID := fmt.Sprintf("agent_%d_doc_%d_chunk_%d", agentID, time.Now().UnixNano(), i) // 生成唯一的ChunkID
		vector := vectors[i]

		metadata := map[string]interface{}{
			"agent_id": agentID,
			"content":  chunkContent,
		}

		err := s.vector.Upsert(ctx, chunkID, vector, metadata)
		if err != nil {
			// 在 MVP 阶段遇到错误直接返回。
			// 生产环境优化点：这里其实应该收集所有失败的片段，支持“部分成功”和重试逻辑。
			return fmt.Errorf("片段 [%d] 入库失败: %w", i, err)
		}
	}

	return nil
}
