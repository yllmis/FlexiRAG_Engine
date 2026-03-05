package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config 数据库配置参数 从YAML中获取	
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// NewPostgresDB 创建并配置数据库连接池
func NewPostgresDB(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 生产级连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)           // 空闲连接池中连接的最大数量
	sqlDB.SetMaxOpenConns(100)          // 数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接可复用的最大时间

	return db, nil
}
