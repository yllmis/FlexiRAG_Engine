package audit

import (
	"context"
	"testing"
	"time"

	"flexirag-engine/internal/core"
)

type mockAuditRepo struct {
	saveN int
}

func (m *mockAuditRepo) Save(ctx context.Context, event core.AuditEvent) error {
	m.saveN++
	return nil
}

func TestAsyncWriter_LogAndDrop(t *testing.T) {
	repo := &mockAuditRepo{}
	writer := NewAsyncWriter(repo, 1)

	for i := 0; i < 50; i++ {
		writer.Log(core.AuditEvent{EventType: "x"})
	}
	time.Sleep(100 * time.Millisecond)
	enq, drop, _ := writer.Stats()
	if enq == 0 {
		t.Fatal("期望有事件入队")
	}
	if drop == 0 {
		t.Fatal("期望在小队列下出现丢弃")
	}
	if repo.saveN == 0 {
		t.Fatal("期望 worker 消费并写入")
	}
}
