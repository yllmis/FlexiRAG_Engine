package ports

import (
	"context"
	"flexirag-engine/internal/core"
)

// AuditRepository 定义审计落库标准
type AuditRepository interface {
	Save(ctx context.Context, event core.AuditEvent) error
}

// AuditLogger 定义审计记录器标准
type AuditLogger interface {
	Log(event core.AuditEvent)
}