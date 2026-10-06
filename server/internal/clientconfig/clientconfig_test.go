package clientconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clearCredentialEnv pins both credential variables to empty so tests
// that exercise file behavior are not affected by the ambient process
// environment. t.Setenv also restores the prior values afterwards.
func clearCredentialEnv(t *testing.T) {
	t.Helper()
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
}

func TestLoadMissingFileIsZeroConfig(t *testing.T) {
	clearCredentialEnv(t)
	path := filepath.Join(t.TempDir(), "insideout", "config.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(missing file) error = %v, want nil", err)
	}
	if cfg != (Config{}) {
		t.Fatalf("Load(missing file) = %+v, want zero Config", cfg)
	}
}

func TestLoadProcessEnvOverridesFilePerField(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	onDisk := Config{APIBase: "https://file.example/api/v1", Token: "file-token"}
	if err := Save(file, onDisk); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cases := []struct {
		name     string
		apiEnv   string
		tokenEnv string
		want     Config
	}{
		{
			name:     "env overrides both fields",
			apiEnv:   "https://env.example/api/v1",
			tokenEnv: "env-token",
			want:     Config{APIBase: "https://env.example/api/v1", Token: "env-token"},
		},
		{
			name:     "api env only overrides APIBase",
			apiEnv:   "https://env.example/api/v1",
			tokenEnv: "",
			want:     Config{APIBase: "https://env.example/api/v1", Token: "file-token"},
		},
		{
			name:     "token env only overrides Token",
			apiEnv:   "",
			tokenEnv: "env-token",
			want:     Config{APIBase: "https://file.example/api/v1", Token: "env-token"},
		},
		{
			name:     "empty env keeps file values",
			apiEnv:   "",
			tokenEnv: "",
			want:     onDisk,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("INSIDEOUT_API", tc.apiEnv)
			t.Setenv("INSIDEOUT_TOKEN", tc.tokenEnv)
			got, err := Load(file)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Load = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadMalformedJSONIsDescriptiveError(t *testing.T) {
	clearCredentialEnv(t)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg, err := Load(path)
	if err == nil {
		t.Fatal("Load(malformed JSON) error = nil, want an error")
	}
	if cfg != (Config{}) {
		t.Errorf("Load(malformed JSON) = %+v, want zero Config on error", cfg)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the config path %q", err, path)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	clearCredentialEnv(t)
	path := filepath.Join(t.TempDir(), "insideout", "config.json")
	want := Config{APIBase: "https://insideout.example/api/v1", Token: "round-trip-token"}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestSaveFileModeIs0600(t *testing.T) {
	clearCredentialEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "insideout", "config.json")
	if err := Save(path, Config{APIBase: "https://insideout.example/api/v1", Token: "mode-token"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("config file mode = %04o, want 0600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat config dir: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Errorf("Save did not create parent directory %s", filepath.Dir(path))
	}
}

func TestDefaultPathJoinsUserConfigDir(t *testing.T) {
	// Pin HOME so os.UserConfigDir is hermetic (on darwin it resolves
	// $HOME/Library/Application Support; XDG_CONFIG_HOME is not honored).
	t.Setenv("HOME", t.TempDir())
	userDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir: %v", err)
	}
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := filepath.Join(userDir, "insideout", "config.json")
	if got != want {
		t.Fatalf("DefaultPath = %q, want %q", got, want)
	}
}

// The common case for every CLI verb: no config file exists yet and
// the operator points the binary with INSIDEOUT_API/INSIDEOUT_TOKEN.
// The environment must win even when there is no file to override —
// found by the join phase wiring (2026-10-07).
func TestLoadMissingFileStillAppliesEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", "config.json")
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(missing) error: %v", err)
	}
	if cfg.APIBase != "https://env.example/api/v1" {
		t.Errorf("APIBase = %q, want the env value", cfg.APIBase)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want the env value", cfg.Token)
	}
}
