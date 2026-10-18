package apiclient

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPConfigUsesDefaultBaseWhenEnvUnset(t *testing.T) {
	script := filepath.Join("..", "..", "..", "scripts", "mcp-config.sh")
	cmd := exec.Command("bash", script, "-q")
	// An absent config file keeps the result off the machine's real one
	// (Linux reads $XDG_CONFIG_HOME, macOS ~/Library/Application Support).
	cmd.Env = append(withoutAPIEnv(os.Environ()),
		"INSIDEOUT_CONFIG_FILE="+filepath.Join(t.TempDir(), "absent.json"))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mcp-config.sh: %v", err)
	}
	var body struct {
		MCPServers map[string]struct {
			Env map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	env := body.MCPServers["insideout"].Env
	if env["INSIDEOUT_API"] != DefaultBase {
		t.Fatalf("INSIDEOUT_API = %q, want %s", env["INSIDEOUT_API"], DefaultBase)
	}
	if _, ok := env["INSIDEOUT_TOKEN"]; ok {
		t.Fatal("unset token must be omitted, not sent as a placeholder")
	}
}

func withoutAPIEnv(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		if strings.HasPrefix(kv, "INSIDEOUT_API=") || strings.HasPrefix(kv, "INSIDEOUT_TOKEN=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
