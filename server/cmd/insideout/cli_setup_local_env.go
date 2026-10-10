package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findRepoRoot walks up from the working directory to a checkout
// (docker-compose.yml + .env.example + server/go.mod).
// INSIDEOUT_SETUP_REPO_ROOT overrides — the test affordance — but the
// override must still be a real checkout.
func findRepoRoot() (string, error) {
	if v := os.Getenv("INSIDEOUT_SETUP_REPO_ROOT"); v != "" {
		if !repoMarker(v) {
			return "", fmt.Errorf("INSIDEOUT_SETUP_REPO_ROOT=%s is not a repository checkout", v)
		}
		return v, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if repoMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no repository checkout found above %s", dir)
		}
		dir = parent
	}
}

func repoMarker(dir string) bool {
	for _, name := range []string{"docker-compose.yml", ".env.example", filepath.Join("server", "go.mod")} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

// readEnvValues reads KEY=VALUE lines (last wins, comments skipped).
// Values feed machine-to-machine assembly (DSN construction, child
// env) and are never printed.
func readEnvValues(path string) map[string]string {
	vals := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return vals
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ") // dotenv-tolerant
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" {
			vals[k] = v
		}
	}
	return vals
}

// jwtNeedsRotation mirrors scripts/setup.sh: placeholder, or shorter
// than 32 characters, is not a usable secret.
func jwtNeedsRotation(s string) bool {
	return s == "" || s == "change_me_to_a_long_random_string" || len(s) < 32
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// childEnv returns an environ for the server child: the current
// environment with the setup-written values overriding, plus
// INSIDEOUT_COOKIE_SECURE=0 because a local server serves plain http
// (the value .env.example documents for exactly this case).
func childEnv(environ []string, vals map[string]string) []string {
	override := map[string]string{
		"DATABASE_URL":            vals["DATABASE_URL"],
		"INSIDEOUT_JWT_SECRET":    vals["INSIDEOUT_JWT_SECRET"],
		"INSIDEOUT_COOKIE_SECURE": "0",
	}
	var out []string
	for _, kv := range environ {
		key, _, ok := strings.Cut(kv, "=")
		if ok && override[key] != "" {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range override {
		if v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}
