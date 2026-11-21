package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// signup_mode reads the public /auth/config, the CLI `signup-mode` verb with no
// argument. Setting the mode is an admin route no MCP token reaches (PARA-07).
func TestSignupModeTool(t *testing.T) {
	var paths []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Write([]byte(`{"signupMode":"invitation_only"}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{toolCall(1, "signup_mode", `{}`)})
	if r := res[1]; r.Result.IsError || len(r.Result.Content) == 0 || !strings.Contains(r.Result.Content[0].Text, "invitation_only") {
		t.Errorf("signup_mode = %+v, want the current mode", r)
	}
	if len(paths) != 1 || paths[0] != "GET /auth/config" {
		t.Errorf("signup_mode sent %v, want GET /auth/config", paths)
	}
}
