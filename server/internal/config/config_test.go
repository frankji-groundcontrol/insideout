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

func TestLoadRejectsMailKeyWithoutPublicURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_MAIL_API_KEY", "re-test-key")
	t.Setenv("INSIDEOUT_PUBLIC_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected mail key without INSIDEOUT_PUBLIC_URL to fail")
	}
}

func TestLoadAcceptsMailKeyWithPublicURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_MAIL_API_KEY", "re-test-key")
	t.Setenv("INSIDEOUT_PUBLIC_URL", "https://app.example.com")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PublicURL != "https://app.example.com" {
		t.Fatalf("PublicURL = %q", cfg.PublicURL)
	}
}

func TestLoadMailDevBodyDefaultsFalse(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_MAIL_DEV_BODY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MailDevBody {
		t.Fatal("MailDevBody should default to false")
	}
}

func TestLoadMailDevBodyParsesOne(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("INSIDEOUT_MAIL_DEV_BODY", "1")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.MailDevBody {
		t.Fatal(`MailDevBody should be true with INSIDEOUT_MAIL_DEV_BODY=1`)
	}
}

// INSIDEOUT_DEV_CORS=1 mounts a middleware that reflects any Origin with
// credentials, so Load refuses it unless the app's public origin is local —
// otherwise one copied dev .env hands a hosted deployment to any website.
func TestLoadRejectsDevCORSBesideNonLocalPublicURL(t *testing.T) {
	for _, publicURL := range []string{
		"https://app.example.com",
		"http://192.168.1.20:5173",
		"http://localhost.evil.com",
		"localhost:5173", // no scheme: not a loopback URL
		"http://[::1",    // unclosed bracket: url.Parse fails, so not local
	} {
		t.Run(publicURL, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
			t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
			t.Setenv("INSIDEOUT_DEV_CORS", "1")
			t.Setenv("INSIDEOUT_PUBLIC_URL", publicURL)
			_, err := Load()
			if err == nil {
				t.Fatal("expected INSIDEOUT_DEV_CORS=1 beside a non-local INSIDEOUT_PUBLIC_URL to fail")
			}
			if !strings.Contains(err.Error(), "INSIDEOUT_DEV_CORS") {
				t.Fatalf("error should name INSIDEOUT_DEV_CORS, got: %v", err)
			}
		})
	}
}

func TestLoadAcceptsDevCORSBesideLocalPublicURL(t *testing.T) {
	cases := []struct {
		name      string
		devCORS   string
		publicURL string // "" = unset, which takes the localhost default
		want      bool   // DevPermissiveCORS
	}{
		{"unset takes the localhost default", "1", "", true},
		{"localhost", "1", "http://localhost:5173", true},
		{"loopback v4", "1", "http://127.0.0.1:5173", true},
		{"loopback v6", "1", "http://[::1]:5173", true},
		{"host names are case-insensitive", "1", "http://LocalHost:5173", true},
		{"flag off beside a hosted origin", "", "https://app.example.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
			t.Setenv("INSIDEOUT_JWT_SECRET", strings.Repeat("s", 32))
			t.Setenv("INSIDEOUT_DEV_CORS", tc.devCORS)
			t.Setenv("INSIDEOUT_PUBLIC_URL", tc.publicURL)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.DevPermissiveCORS != tc.want {
				t.Fatalf("DevPermissiveCORS = %v, want %v", cfg.DevPermissiveCORS, tc.want)
			}
		})
	}
}
