package config

import (
	"strings"
	"testing"
)

func TestListenAddrDefault(t *testing.T) {
	t.Setenv("INSIDEOUT_ADDR", "")
	t.Setenv("PORT", "")
	if got := listenAddr(); got != ":8080" {
		t.Fatalf("listenAddr() = %q, want :8080", got)
	}
}

func TestListenAddrHonorsPORT(t *testing.T) {
	t.Setenv("INSIDEOUT_ADDR", "")
	t.Setenv("PORT", "9090")
	if got := listenAddr(); got != ":9090" {
		t.Fatalf("listenAddr() = %q, want :9090", got)
	}
}

func TestListenAddrPrefersExplicitAddr(t *testing.T) {
	t.Setenv("INSIDEOUT_ADDR", ":7777")
	t.Setenv("PORT", "9090")
	if got := listenAddr(); got != ":7777" {
		t.Fatalf("listenAddr() = %q, want :7777", got)
	}
}

func TestLoadOptionalOwnerURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://insideout_app:p@localhost/db")
	t.Setenv("DATABASE_OWNER_URL", "postgres://insideout_owner:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseOwnerURL == "" {
		t.Fatal("DatabaseOwnerURL empty")
	}
}

func TestLoadReadsLLMEnvAndDefaultsSchema(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_LLM_BASE_URL", "https://gateway.example/v1")
	t.Setenv("INSIDEOUT_LLM_API_KEY", "sk-test")
	t.Setenv("INSIDEOUT_LLM_MODEL", "demo-model")
	t.Setenv("INSIDEOUT_LLM_SCHEMA", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "old-must-be-ignored")
	t.Setenv("ANTHROPIC_BASE_URL", "https://old.example")
	t.Setenv("AI_MODEL", "old-model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIBaseURL != "https://gateway.example/v1" {
		t.Fatalf("AIBaseURL = %q", cfg.AIBaseURL)
	}
	if cfg.AIAuthToken != "sk-test" {
		t.Fatalf("AIAuthToken = %q", cfg.AIAuthToken)
	}
	if cfg.AIModel != "demo-model" {
		t.Fatalf("AIModel = %q", cfg.AIModel)
	}
	if cfg.AISchema != "messages" {
		t.Fatalf("AISchema = %q, want messages", cfg.AISchema)
	}
}

func TestLoadAcceptsShortLLMModelAndSchemaNames(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_LLM_MODEL", "")
	t.Setenv("INSIDEOUT_LLM_SCHEMA", "")
	t.Setenv("INSIDE_LLM_MODEL", "short-model")
	t.Setenv("INSIDE_LLM_SCHEMA", "responses")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIModel != "short-model" || cfg.AISchema != "responses" {
		t.Fatalf("model=%q schema=%q", cfg.AIModel, cfg.AISchema)
	}
}

func TestLoadRejectsUnknownLLMSchema(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_LLM_SCHEMA", "completions")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid schema to fail")
	}
}
