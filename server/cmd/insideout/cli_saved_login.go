package main

// The saved-login file helpers: read the login saved for one server, and
// forget it on logout or account deletion.

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// savedLoginFor is the config file's login for base, or a zero config when
// nothing was saved for that server (missing file, or another server's login).
func savedLoginFor(base string) (clientconfig.Config, error) {
	path, err := clientconfig.DefaultPath()
	if err != nil {
		return clientconfig.Config{}, nil
	}
	cfg, err := clientconfig.Load(path)
	if err != nil || !basesMatch(base, savedBase(cfg.APIBase)) {
		return clientconfig.Config{}, err
	}
	return cfg, nil
}

// forgetSavedLogin removes the saved login for base. The file goes when it
// holds nothing else; any other field stays and only the token fields clear.
func forgetSavedLogin(base string) error {
	path, err := clientconfig.DefaultPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	saved, _ := raw["api_base"].(string)
	if !basesMatch(base, savedBase(saved)) {
		return nil
	}
	extra := false
	for key := range raw {
		if key != "api_base" && key != "token" && key != "refresh_token" {
			extra = true
		}
	}
	if !extra {
		return os.Remove(path)
	}
	delete(raw, "token")
	delete(raw, "refresh_token")
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func savedBase(apiBase string) string {
	if apiBase == "" {
		return apiclient.DefaultBase
	}
	return apiBase
}

func basesMatch(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}
