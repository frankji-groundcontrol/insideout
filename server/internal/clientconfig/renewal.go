package clientconfig

import (
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// The saved session's side of apiclient.Renewal. Both hooks act on the file
// only while it still holds a login for the server these Creds talk to: a
// `insideout login --api ...` in another terminal may have pointed it
// elsewhere, and that newer login must not be overwritten or adopted.

// holds reports whether cfg is a login for c's server.
func (c Creds) holds(cfg Config) bool {
	return sameBase(firstNonEmpty(cfg.APIBase, apiclient.DefaultBase), c.Base)
}

// reload returns the pair currently stored for c's server.
func (c Creds) reload() (access, refresh string, ok bool) {
	cfg, err := Load(c.path)
	if err != nil || !c.holds(cfg) || cfg.RefreshToken == "" {
		return "", "", false
	}
	return cfg.Token, cfg.RefreshToken, true
}

// save stores a rotated pair next to the rest of the saved login. The server
// has already retired the old refresh token, so a failure here costs the user
// a fresh sign-in: say so, loudly, once.
func (c Creds) save(access, refresh string) {
	cfg, err := Load(c.path)
	if err == nil {
		if !c.holds(cfg) {
			return
		}
		cfg.Token, cfg.RefreshToken = access, refresh
		err = Save(c.path, cfg)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "insideout: renewed the session but could not save it: %v\n"+
			"insideout: run `insideout login` before this session runs out\n", err)
	}
}
