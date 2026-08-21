// Package config parses and fail-fast validates environment configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	// DatabaseOwnerURL is insideout_owner (NOSUPERUSER) for DDL/migrate.
	// Empty means Migrate uses DatabaseURL, which must already be that owner.
	DatabaseOwnerURL string
	JWTSecret        string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	AIBaseURL        string
	AIAuthToken      string
	AIModel          string
	// AISchema is "messages" (Anthropic Messages wire format at
	// {base}/messages) or "responses" (OpenAI Responses at {base}/responses).
	// The operator includes any /v1 prefix on AIBaseURL.
	AISchema          string
	DevPermissiveCORS bool
	// CORSOrigins is an exact Origin allow-list for Flutter web (and any
	// other browser client that is not same-origin). The token "localhost"
	// matches http(s) localhost / 127.0.0.1 with any port. Empty means no
	// allow-list CORS (Nuxt same-origin needs none).
	CORSOrigins []string
	// CookieSecure controls the Secure attribute on auth cookies. Defaults
	// to true; set INSIDEOUT_COOKIE_SECURE=0 for plain-http local dev
	// where the browser wouldn't otherwise store the cookie.
	// 控制认证 cookie 的 Secure 属性，默认 true；纯 HTTP 本地开发时设为
	// INSIDEOUT_COOKIE_SECURE=0，否则浏览器不会存储该 cookie。
	CookieSecure bool
	// GithubWebhookSecret verifies GitHub App deliveries (HMAC-SHA256,
	// X-Hub-Signature-256). Empty disables POST /api/v1/hooks/github.
	GithubWebhookSecret string
	// GithubAppID + GithubPrivateKey mint installation access tokens
	// (server-to-server; no user OAuth). The key is the app's .pem,
	// \n-escaped for env storage; GithubPrivateKeyFile (the _FILE
	// variant) wins when set.
	GithubAppID          string
	GithubPrivateKey     string
	GithubPrivateKeyFile string
}

func Load() (*Config, error) {
	c := &Config{
		Addr:                 listenAddr(),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseOwnerURL:     os.Getenv("DATABASE_OWNER_URL"),
		JWTSecret:            os.Getenv("INSIDEOUT_JWT_SECRET"),
		AIBaseURL:            getenv("INSIDEOUT_LLM_BASE_URL", "https://api.anthropic.com/v1"),
		AIAuthToken:          os.Getenv("INSIDEOUT_LLM_API_KEY"),
		AIModel:              getenvFirst("claude-sonnet-4-20250514", "INSIDEOUT_LLM_MODEL", "INSIDE_LLM_MODEL"),
		AISchema:             getenvFirst("messages", "INSIDEOUT_LLM_SCHEMA", "INSIDE_LLM_SCHEMA"),
		GithubWebhookSecret:  os.Getenv("INSIDEOUT_GH_WEBHOOK_SECRET"),
		GithubAppID:          os.Getenv("INSIDEOUT_GH_APP_ID"),
		GithubPrivateKey:     os.Getenv("INSIDEOUT_GH_PRIVATE_KEY"),
		GithubPrivateKeyFile: os.Getenv("INSIDEOUT_GH_PRIVATE_KEY_FILE"),
	}

	var err error
	if c.AccessTTL, err = parseDuration("INSIDEOUT_ACCESS_TTL", "15m"); err != nil {
		return nil, err
	}
	if c.RefreshTTL, err = parseDuration("INSIDEOUT_REFRESH_TTL", "720h"); err != nil {
		return nil, err
	}
	c.DevPermissiveCORS = os.Getenv("INSIDEOUT_DEV_CORS") == "1"
	c.CORSOrigins = splitCSV(os.Getenv("INSIDEOUT_CORS_ORIGINS"))
	c.CookieSecure = os.Getenv("INSIDEOUT_COOKIE_SECURE") != "0"

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("config: INSIDEOUT_JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("config: INSIDEOUT_JWT_SECRET must be at least 32 characters")
	}
	switch c.AISchema {
	case "messages", "responses":
	default:
		return nil, fmt.Errorf("config: INSIDEOUT_LLM_SCHEMA must be messages or responses, got %q", c.AISchema)
	}

	return c, nil
}

func getenv(key, fallback string) string {
	return getenvFirst(fallback, key)
}

func getenvFirst(fallback string, keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}

// listenAddr prefers INSIDEOUT_ADDR, then a platform PORT (Railway/Heroku),
// then :8080. PORT is digits only; a leading colon is accepted if present.
func listenAddr() string {
	if v := os.Getenv("INSIDEOUT_ADDR"); v != "" {
		return v
	}
	if p := os.Getenv("PORT"); p != "" {
		if p[0] == ':' {
			return p
		}
		return ":" + p
	}
	return ":8080"
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := getenv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s %q: %w", key, raw, err)
	}
	return d, nil
}
