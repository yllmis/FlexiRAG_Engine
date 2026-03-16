package audit

import (
	"context"
	"log"
	"sync/atomic"

	"flexirag-engine/internal/core"
)

var _ core.AuditLogger = (*AsyncWriter)(nil)

type AsyncWriter struct {
	repo  core.AuditRepository
	queue chan core.AuditEvent
	dropN atomic.Uint64
	errN  atomic.Uint64
	enqN  atomic.Uint64
}

func NewAsyncWriter(repo core.AuditRepository, queueSize int) *AsyncWriter {
	if queueSize <= 0 {
		queueSize = 1024
	}
	w := &AsyncWriter{
		repo:  repo,
		queue: make(chan core.AuditEvent, queueSize),
	}
	go w.worker()
	return w
}

func (w *AsyncWriter) Log(event core.AuditEvent) {
	select {
	case w.queue <- event:
		w.enqN.Add(1)
	default:
		w.dropN.Add(1)
		log.Printf("审计队列已满，丢弃事件 event_type=%s request_id=%s", event.EventType, event.RequestID)
	}
}

func (w *AsyncWriter) worker() {
	for event := range w.queue {
		if err := w.repo.Save(context.Background(), event); err != nil {
			w.errN.Add(1)
			log.Printf("审计写入失败 event_type=%s request_id=%s err=%v", event.EventType, event.RequestID, err)
		}
	}
}

func (w *AsyncWriter) Stats() (enqueued, dropped, failed uint64) {
	return w.enqN.Load(), w.dropN.Load(), w.errN.Load()
}
