package clientconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// Creds is one resolved set of client credentials: the API base to talk to
// and the bearer token for exactly that base (empty means signed out).
type Creds struct {
	Base  string
	Token string
}

// Resolve is the one place a client surface decides which server it talks to
// and as whom.
//
// Base: flagAPI, else INSIDEOUT_API, else the saved login's api_base, else
// apiclient.DefaultBase.
//
// Token: INSIDEOUT_TOKEN when set — an explicit credential for whichever base
// is in effect — else the saved token, but only when the effective base is the
// one the token was saved for (a config without api_base was saved for the
// hosted default). A token saved for server A is never sent to server B.
//
// A config file that exists but cannot be read is an error, never a silent
// fall back to the hosted default. A missing file, or no config directory at
// all (no HOME), simply means nothing was saved.
func Resolve(flagAPI string) (Creds, error) {
	var cfg Config
	if path, err := DefaultPath(); err == nil {
		if cfg, err = Load(path); err != nil {
			return Creds{}, fmt.Errorf("%w (fix or delete the file, then sign in again with `insideout login`)", err)
		}
	}
	saved := firstNonEmpty(cfg.APIBase, apiclient.DefaultBase)
	base := firstNonEmpty(flagAPI, os.Getenv("INSIDEOUT_API"), saved)
	token := os.Getenv("INSIDEOUT_TOKEN")
	if token == "" && sameBase(base, saved) {
		token = cfg.Token
	}
	return Creds{Base: base, Token: token}, nil
}

// Client returns an API client for these credentials.
func (c Creds) Client() *apiclient.Client {
	cl := apiclient.New(c.Base)
	cl.SetToken(c.Token)
	return cl
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func sameBase(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}
