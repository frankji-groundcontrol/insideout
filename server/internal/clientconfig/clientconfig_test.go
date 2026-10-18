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
