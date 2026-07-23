// Package config parses and fail-fast validates environment configuration.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr              string
	DatabaseURL       string
	JWTSecret         string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	AIBaseURL         string
	AIAuthToken       string
	AIModel           string
	DevPermissiveCORS bool
	// CookieSecure controls the Secure attribute on auth cookies. Defaults
	// to true; set INSIDEOUT_COOKIE_SECURE=0 for plain-http local dev
	// where the browser wouldn't otherwise store the cookie.
	// 控制认证 cookie 的 Secure 属性，默认 true；纯 HTTP 本地开发时设为
	// INSIDEOUT_COOKIE_SECURE=0，否则浏览器不会存储该 cookie。
	CookieSecure bool
}

func Load() (*Config, error) {
	c := &Config{
		Addr:        getenv("INSIDEOUT_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("INSIDEOUT_JWT_SECRET"),
		AIBaseURL:   os.Getenv("ANTHROPIC_BASE_URL"),
		AIAuthToken: os.Getenv("ANTHROPIC_AUTH_TOKEN"),
		AIModel:     getenv("AI_MODEL", "claude-sonnet-4-20250514"),
	}

	var err error
	if c.AccessTTL, err = parseDuration("INSIDEOUT_ACCESS_TTL", "15m"); err != nil {
		return nil, err
	}
	if c.RefreshTTL, err = parseDuration("INSIDEOUT_REFRESH_TTL", "720h"); err != nil {
		return nil, err
	}
	c.DevPermissiveCORS = os.Getenv("INSIDEOUT_DEV_CORS") == "1"
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

	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := getenv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s %q: %w", key, raw, err)
	}
	return d, nil
}
