// Package clientconfig resolves InsideOut client credentials from a
// single place. Resolution precedence is per field: the process
// environment (INSIDEOUT_API, INSIDEOUT_TOKEN) overrides the per-user
// config file, and only a non-empty variable overrides. Every CLI verb
// and the MCP server resolve credentials through this package rather
// than reading the environment or the file themselves.
package clientconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the persisted client configuration.
type Config struct {
	APIBase string `json:"api_base,omitempty"`
	Token   string `json:"token,omitempty"`
}

// DefaultPath returns the per-user config file location:
// os.UserConfigDir() + "/insideout/config.json".
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "insideout", "config.json"), nil
}

// Load reads the config at path. A missing file yields a zero Config
// and a nil error; malformed JSON yields a descriptive error naming
// the path. Per field, a non-empty INSIDEOUT_API or INSIDEOUT_TOKEN in
// the process environment overrides the value loaded from the file —
// including when no file exists at all, which is the common case for
// a CLI pointed by environment alone.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config %s: %w", path, err)
		}
	case os.IsNotExist(err):
		// no file: fall through so env overrides still apply
	default:
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if v := os.Getenv("INSIDEOUT_API"); v != "" {
		cfg.APIBase = v
	}
	if v := os.Getenv("INSIDEOUT_TOKEN"); v != "" {
		cfg.Token = v
	}
	return cfg, nil
}

// Save writes cfg to path. The parent directory is created with 0700,
// the JSON is written to a temp file in the same directory, the temp
// file is chmod 0600, and only then is it renamed over the target so
// the config file never appears partially written or world-readable.
func Save(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	f, err := os.CreateTemp(dir, ".insideout-config-*")
	if err != nil {
		return fmt.Errorf("create temp config in %s: %w", dir, err)
	}
	tmp := f.Name()
	// No-op once rename has moved the file; cleans up on error paths.
	defer os.Remove(tmp)

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write temp config %s: %w", tmp, err)
	}
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return fmt.Errorf("chmod temp config %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp config %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}
