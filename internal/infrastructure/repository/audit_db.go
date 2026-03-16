package repository

import (
	"context"
	"time"

	"flexirag-engine/internal/core"

	"gorm.io/gorm"
)

var _ core.AuditRepository = (*PGAuditRepo)(nil)

type AuditLog struct {
	ID           uint      `gorm:"primaryKey"`
	EventType    string    `gorm:"size:64;index"`
	SubjectID    string    `gorm:"size:128;index"`
	ResourceType string    `gorm:"size:64;index"`
	ResourceID   string    `gorm:"size:128"`
	RequestID    string    `gorm:"size:128;index"`
	Status       string    `gorm:"size:32;index"`
	Message      string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

type PGAuditRepo struct {
	db *gorm.DB
}

func NewPGAuditRepo(db *gorm.DB) (*PGAuditRepo, error) {
	if err := db.AutoMigrate(&AuditLog{}); err != nil {
		return nil, err
	}
	return &PGAuditRepo{db: db}, nil
}

func (r *PGAuditRepo) Save(ctx context.Context, event core.AuditEvent) error {
	row := AuditLog{
		EventType:    event.EventType,
		SubjectID:    event.SubjectID,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		RequestID:    event.RequestID,
		Status:       event.Status,
		Message:      event.Message,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}
