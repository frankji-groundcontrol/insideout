package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 ARCH-02: a saved login that cannot be read is reported by the tool
// that needed it. It is not quietly treated as "signed out" (and the request
// is not sent without the credential the user believes they stored).
func TestToolsReportAnUnreadableConfig(t *testing.T) {
	pinConfigHome(t)
	srv, seen := recordingAPI(t)
	path, err := clientconfig.DefaultPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	res := drive(t, srv.URL, []string{toolCall(1, "whoami", `{}`)})

	r := res[1].Result
	if !r.IsError || len(r.Content) == 0 || !strings.Contains(r.Content[0].Text, "parse config") {
		t.Fatalf("whoami must fail naming the unreadable config, got %+v", r)
	}
	if got := seen(); len(got) != 0 {
		t.Fatalf("the tool sent %q despite the unreadable config", got)
	}
}
