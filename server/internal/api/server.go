package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/mail"
	"github.com/frankji-groundcontrol/insideout/server/internal/presence"
	"github.com/frankji-groundcontrol/insideout/server/internal/progresshub"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
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
	// progress is the in-memory fan-out for roadmap progress SSE.
	progress *progresshub.Hub
	// mail sends transactional email (magic links, password reset);
	// without an API key it logs messages instead of delivering them.
	mail *mail.Sender
	// authThrottle limits brute-force and email-bomb volume on the auth
	// endpoints (per email+IP and per IP); see auth_throttle.go.
	authThrottle *authThrottle
	// streamsCtx ends when shutdown starts (CloseStreams). The passive SSE
	// streams select on it because http.Server.Shutdown never cancels
	// handler contexts.
	streamsCtx    context.Context
	cancelStreams context.CancelFunc
}

func NewServer(st *store.Store, tokens *auth.TokenIssuer, cfg *config.Config, log *slog.Logger, coach Coach, planner RoadmapPlanner) *Server {
	streamsCtx, cancelStreams := context.WithCancel(context.Background())
	s := &Server{store: st, tokens: tokens, cfg: cfg, log: log, coach: coach, planner: planner,
		streamsCtx: streamsCtx, cancelStreams: cancelStreams,
		presence:     presence.New(30*time.Second, nil),
		progress:     progresshub.New(),
		mail:         mail.NewSender(cfg.MailAPIKey, cfg.MailFrom, log, cfg.MailDevBody),
		authThrottle: newAuthThrottle(defaultThrottleConfig(), time.Now),
	}
	if st != nil {
		st.OnRoadmapChange(func(projectID uuid.UUID) {
			s.progress.Publish(projectID.String())
		})
	}
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

// CloseStreams ends the passive SSE streams (presence, roadmap progress).
// Register it with http.Server.RegisterOnShutdown: Shutdown never cancels
// handler contexts, so without it one open tab holds the whole window.
func (s *Server) CloseStreams() { s.cancelStreams() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerAuthRoutes(mux)
	s.registerMeRoutes(mux)
	s.registerAdminRoutes(mux)
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
	// Auth-endpoint rate limiting sits inside recover/logging so throttled
	// requests are still logged and a throttle bug can't 500 the server.
	h = s.withAuthThrottle(h)
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
