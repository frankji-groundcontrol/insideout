package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/presence"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Coach is the subset of internal/agent.Coach the API layer needs; kept as
// an interface here so api doesn't import agent until it's wired in P4,
// and so handler tests can stub it.
type Coach interface {
	HandleMessage(w http.ResponseWriter, r *http.Request, conversationID, userID, workspaceID, content string)
}

type Server struct {
	store   *store.Store
	tokens  *auth.TokenIssuer
	cfg     *config.Config
	log     *slog.Logger
	coach   Coach
	planner RoadmapPlanner
	// ghTokens mints GitHub App installation tokens (nil when the app
	// credentials are absent or the key fails to load).
	ghTokens *github.InstallationTokens
	// presence is the in-memory canvas presence registry.
	presence *presence.Registry
}

func NewServer(st *store.Store, tokens *auth.TokenIssuer, cfg *config.Config, log *slog.Logger, coach Coach, planner RoadmapPlanner) *Server {
	s := &Server{store: st, tokens: tokens, cfg: cfg, log: log, coach: coach, planner: planner,
		presence: presence.New(30*time.Second, nil)}
	// GitHub App installation tokens for guide loading; nil disables the
	// token path (public repos still load unauthenticated).
	if cfg.GithubAppID != "" {
		if key, err := github.LoadPrivateKey(cfg.GithubPrivateKey, cfg.GithubPrivateKeyFile); err != nil {
			log.Warn("github app private key unusable — guide loads will be unauthenticated", "error", err)
		} else {
			s.ghTokens = github.NewInstallationTokens(cfg.GithubAppID, key)
		}
	}
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerAuthRoutes(mux)
	s.registerMeRoutes(mux)
	s.registerWorkspaceRoutes(mux)
	s.registerProjectRoutes(mux)
	s.registerIdeaRoutes(mux)
	s.registerPrdRoutes(mux)
	s.registerConversationRoutes(mux)
	s.registerRoadmapRoutes(mux)
	s.registerGithubRoutes(mux)
	s.registerRoadmapAIRoutes(mux)
	s.registerAgentRoutes(mux)
	s.registerPresenceRoutes(mux)

	mux.HandleFunc("GET /healthz", s.handleHealthz)

	var h http.Handler = mux
	h = s.withMaxBody(h)
	h = s.withRecover(h)
	h = s.withLogging(h)
	h = s.withRequestID(h)
	if s.cfg.DevPermissiveCORS {
		h = s.withDevCORS(h)
	} else if len(s.cfg.CORSOrigins) > 0 {
		h = s.withAllowlistCORS(h)
	}
	return h
}
