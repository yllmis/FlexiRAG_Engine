package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

const defaultConfigPath = "configs/app.yaml"

type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	TimeZone string `yaml:"timezone"`
}

type LLMConfig struct {
	Provider   string `yaml:"provider"`
	APIKey     string `yaml:"api_key"`
	BaseURL    string `yaml:"base_url"`
	ChatModel  string `yaml:"chat_model"`
	EmbedModel string `yaml:"embed_model"`
}

type SecurityConfig struct {
	AdminToken         string `yaml:"admin_token"`
	RateLimitPerMinute int    `yaml:"rate_limit_per_minute"`
	AuditQueueSize     int    `yaml:"audit_queue_size"`
}

func Load(path string) (AppConfig, error) {
	if strings.TrimSpace(path) == "" {
		path = defaultConfigPath
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("解析配置文件失败: %w", err)
	}

	applyDefaults(&cfg)
	overrideByEnv(&cfg)

	if err := validate(cfg); err != nil {
		return AppConfig{}, err
	}

	return cfg, nil
}

// applyDefaults 设置默认值
func applyDefaults(cfg *AppConfig) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if strings.TrimSpace(cfg.Database.Host) == "" {
		cfg.Database.Host = "127.0.0.1"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if strings.TrimSpace(cfg.Database.User) == "" {
		cfg.Database.User = "root"
	}
	if strings.TrimSpace(cfg.Database.DBName) == "" {
		cfg.Database.DBName = "flexirag_db"
	}
	if strings.TrimSpace(cfg.Database.SSLMode) == "" {
		cfg.Database.SSLMode = "disable"
	}
	if strings.TrimSpace(cfg.Database.TimeZone) == "" {
		cfg.Database.TimeZone = "Asia/Shanghai"
	}
	if strings.TrimSpace(cfg.LLM.Provider) == "" {
		cfg.LLM.Provider = "glm"
	}
	if strings.TrimSpace(cfg.LLM.BaseURL) == "" {
		cfg.LLM.BaseURL = "https://open.bigmodel.cn/api/paas/v4/"
	}
	if strings.TrimSpace(cfg.LLM.ChatModel) == "" {
		cfg.LLM.ChatModel = "glm-4-flash"
	}
	if strings.TrimSpace(cfg.LLM.EmbedModel) == "" {
		cfg.LLM.EmbedModel = "embedding-3"
	}
	if strings.TrimSpace(cfg.Security.AdminToken) == "" {
		cfg.Security.AdminToken = "flexirag-secret-123"
	}
	if cfg.Security.RateLimitPerMinute <= 0 {
		cfg.Security.RateLimitPerMinute = 60
	}
	if cfg.Security.AuditQueueSize <= 0 {
		cfg.Security.AuditQueueSize = 1024
	}
}

func overrideByEnv(cfg *AppConfig) {
	overrideString(&cfg.Database.Host, "DB_HOST")
	overrideInt(&cfg.Database.Port, "DB_PORT")
	overrideString(&cfg.Database.User, "DB_USER")
	overrideString(&cfg.Database.Password, "DB_PASSWORD")
	overrideString(&cfg.Database.DBName, "DB_NAME")
	overrideString(&cfg.Database.SSLMode, "DB_SSLMODE")
	overrideString(&cfg.Database.TimeZone, "DB_TIMEZONE")
	overrideInt(&cfg.Server.Port, "SERVER_PORT")
	overrideString(&cfg.LLM.Provider, "LLM_PROVIDER")
	overrideString(&cfg.LLM.APIKey, "OPENAI_API_KEY")
	overrideString(&cfg.LLM.BaseURL, "LLM_BASE_URL")
	overrideString(&cfg.LLM.ChatModel, "LLM_CHAT_MODEL")
	overrideString(&cfg.LLM.EmbedModel, "LLM_EMBED_MODEL")
	overrideString(&cfg.Security.AdminToken, "ADMIN_TOKEN")
	overrideInt(&cfg.Security.RateLimitPerMinute, "RATE_LIMIT_PER_MINUTE")
	overrideInt(&cfg.Security.AuditQueueSize, "AUDIT_QUEUE_SIZE")
}

func overrideString(dst *string, key string) {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		*dst = v
	}
}

func overrideInt(dst *int, key string) {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			*dst = n
		}
	}
}

func validate(cfg AppConfig) error {
	if cfg.Server.Port <= 0 {
		return fmt.Errorf("配置无效: server.port 必须大于 0")
	}
	if strings.TrimSpace(cfg.Database.Password) == "" {
		return fmt.Errorf("配置无效: database.password 不能为空")
	}
	if strings.TrimSpace(cfg.LLM.APIKey) == "" {
		return fmt.Errorf("配置无效: LLM API Key 不能为空，请在配置文件 llm.api_key 或环境变量 OPENAI_API_KEY 中设置")
	}
	if strings.TrimSpace(cfg.Security.AdminToken) == "" {
		return fmt.Errorf("配置无效: security.admin_token 不能为空")
	}
	if cfg.Security.RateLimitPerMinute <= 0 {
		return fmt.Errorf("配置无效: security.rate_limit_per_minute 必须大于 0")
	}
	if cfg.Security.AuditQueueSize <= 0 {
		return fmt.Errorf("配置无效: security.audit_queue_size 必须大于 0")
	}
	return nil
}
