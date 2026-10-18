package clientconfig

import (
	"path/filepath"
	"testing"
)

// Review-4 ARCH-02: Load reports what the file says and nothing else. The
// environment is applied by Resolve, per credential, together with the check
// that a saved token still belongs to the base it is sent to — a per-field
// overlay inside Load pairs one server's token with another server's base.
func TestLoadReadsOnlyTheFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	onDisk := Config{APIBase: "https://file.example/api/v1", Token: "file-token"}
	if err := Save(file, onDisk); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv("INSIDEOUT_API", "https://env.example/api/v1")
	t.Setenv("INSIDEOUT_TOKEN", "env-token")

	got, err := Load(file)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != onDisk {
		t.Fatalf("Load = %+v, want exactly the file's %+v", got, onDisk)
	}

	got, err = Load(filepath.Join(t.TempDir(), "absent", "config.json"))
	if err != nil || got != (Config{}) {
		t.Fatalf("Load(missing file) = %+v, %v; want the zero Config whatever the environment says", got, err)
	}
}
