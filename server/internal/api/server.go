package api

import (
	"log/slog"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
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
}

func NewServer(st *store.Store, tokens *auth.TokenIssuer, cfg *config.Config, log *slog.Logger, coach Coach, planner RoadmapPlanner) *Server {
	return &Server{store: st, tokens: tokens, cfg: cfg, log: log, coach: coach, planner: planner}
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

	mux.HandleFunc("GET /healthz", s.handleHealthz)

	var h http.Handler = mux
	h = s.withRecover(h)
	h = s.withLogging(h)
	h = s.withRequestID(h)
	if s.cfg.DevPermissiveCORS {
		h = s.withDevCORS(h)
	}
	return h
}
