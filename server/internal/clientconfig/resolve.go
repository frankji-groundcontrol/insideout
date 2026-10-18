package clientconfig

import "github.com/frankji-groundcontrol/insideout/server/internal/apiclient"

// Creds is the credential one client invocation runs with.
type Creds struct {
	Base  string // effective API base
	Token string // bearer token; empty means signed out
}

// Resolve is the single place a client decides which API it talks to and as
// whom. The base is the flag value, then INSIDEOUT_API, then the saved
// login's api_base, then the hosted default (apiclient.DefaultBase). Every
// CLI verb and the MCP server resolve through it.
func Resolve(flagAPI string) Creds {
	var cfg Config
	if path, err := DefaultPath(); err == nil {
		cfg, _ = Load(path)
	}
	base := flagAPI
	if base == "" {
		base = cfg.APIBase
	}
	if base == "" {
		base = apiclient.DefaultBase
	}
	return Creds{Base: base, Token: cfg.Token}
}

// Client returns an API client for the resolved credential.
func (c Creds) Client() *apiclient.Client {
	cl := apiclient.New(c.Base)
	cl.SetToken(c.Token)
	return cl
}
