package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ConfigFileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	content := []byte(`server:
  port: 18080
database:
  host: 127.0.0.1
  port: 5432
  user: root
  password: pass
  dbname: flexirag_db
llm:
  api_key: test-key
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load 返回错误: %v", err)
	}

	if cfg.Server.Port != 18080 {
		t.Fatalf("server.port 期望 18080，实际 %d", cfg.Server.Port)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Fatalf("sslmode 默认值错误，实际 %s", cfg.Database.SSLMode)
	}
	if cfg.Database.TimeZone != "Asia/Shanghai" {
		t.Fatalf("timezone 默认值错误，实际 %s", cfg.Database.TimeZone)
	}
	if cfg.LLM.BaseURL == "" || cfg.LLM.ChatModel == "" || cfg.LLM.EmbedModel == "" {
		t.Fatal("LLM 默认配置未生效")
	}
	if cfg.Security.AdminToken == "" {
		t.Fatal("security.admin_token 默认值未生效")
	}
	if cfg.Security.RateLimitPerMinute <= 0 || cfg.Security.AuditQueueSize <= 0 {
		t.Fatal("security 默认阈值未生效")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	content := []byte(`server:
  port: 8080
database:
  host: 127.0.0.1
  port: 5432
  user: root
  password: pass
  dbname: flexirag_db
llm:
  api_key: test-key
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("OPENAI_API_KEY", "override-key")
	t.Setenv("ADMIN_TOKEN", "custom-admin-token")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load 返回错误: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("环境变量 SERVER_PORT 覆盖失败，实际 %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.internal" {
		t.Fatalf("环境变量 DB_HOST 覆盖失败，实际 %s", cfg.Database.Host)
	}
	if cfg.LLM.APIKey != "override-key" {
		t.Fatalf("环境变量 OPENAI_API_KEY 覆盖失败，实际 %s", cfg.LLM.APIKey)
	}
	if cfg.Security.AdminToken != "custom-admin-token" {
		t.Fatalf("环境变量 ADMIN_TOKEN 覆盖失败，实际 %s", cfg.Security.AdminToken)
	}
}

func TestLoad_RequireAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	content := []byte(`server:
  port: 8080
database:
  host: 127.0.0.1
  port: 5432
  user: root
  password: pass
  dbname: flexirag_db
llm:
  api_key: ""
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("期望因 API Key 缺失而报错，实际未报错")
	}
}
