// Command insideout runs the InsideOut API server. Subcommands:
//
//	insideout            run the HTTP server
//	insideout migrate     apply pending SQL migrations and exit
//	insideout seed        create demo data for local development
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/api"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("open store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	switch cmd := args(); cmd {
	case "migrate":
		runMigrate(ctx, log, st)
	case "seed":
		runSeed(ctx, log, st)
	case "":
		runServe(ctx, log, st, cfg)
	default:
		log.Error("unknown subcommand", "command", cmd)
		os.Exit(1)
	}
}

func args() string {
	if len(os.Args) < 2 {
		return ""
	}
	return os.Args[1]
}

func runMigrate(ctx context.Context, log *slog.Logger, st *store.Store) {
	applied, err := st.Migrate(ctx)
	if err != nil {
		log.Error("migrate", "error", err)
		os.Exit(1)
	}
	if len(applied) == 0 {
		log.Info("migrate: nothing to do, already up to date")
		return
	}
	log.Info("migrate: applied", "files", applied)
}

func runServe(ctx context.Context, log *slog.Logger, st *store.Store, cfg *config.Config) {
	if _, err := st.Migrate(ctx); err != nil {
		log.Error("migrate on startup", "error", err)
		os.Exit(1)
	}

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTTL)

	var streamer agent.ChatStreamer
	var planner agent.RoadmapPlanner
	if cfg.AIAuthToken == "" {
		log.Info("ANTHROPIC_AUTH_TOKEN not set — using offline template-reply coach")
		streamer = agent.NewTemplateStreamer()
		planner = agent.NewTemplateRoadmapPlanner()
	} else {
		s, err := agent.NewAnthropicStreamer(cfg.AIBaseURL, cfg.AIAuthToken, cfg.AIModel)
		if err != nil {
			log.Error("create anthropic streamer", "error", err)
			os.Exit(1)
		}
		streamer = s
		planner = agent.NewAnthropicRoadmapPlanner(s)
		checkModelAtStartup(ctx, log, s, cfg.AIModel)
	}
	coach := agent.New(st, streamer, log)

	srv := api.NewServer(st, tokens, cfg, log, coach, planner)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go runReaper(ctx, log, st)

	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
	}
}

// checkModelAtStartup turns BUG-009's silent model_not_found (surfaced
// only at the first real user request) into a loud boot-time signal.
// Gateways sometimes don't implement /v1/models faithfully, so a check
// failure only warns — it never blocks startup.
func checkModelAtStartup(ctx context.Context, log *slog.Logger, s *agent.AnthropicStreamer, model string) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ok, available, err := s.CheckModel(checkCtx)
	if err != nil {
		log.Warn("could not verify AI_MODEL against the provider's /v1/models — proceeding anyway", "model", model, "error", err)
		return
	}
	if !ok {
		log.Warn("AI_MODEL is not in the provider's model list — the coach will fail at the first request", "model", model, "available", available)
	}
}

// reaperInterval and reaperStaleAfter — see
// docs/plans/2026-07-21-prd-agent-harness/plan.md §5.3.
const (
	reaperInterval   = 5 * time.Minute
	reaperStaleAfter = 10 * time.Minute
)

// runReaper marks ai_runs stuck in pending/running with no heartbeat for
// reaperStaleAfter as failed — fixes contradictory run states and dead
// message placeholders left by a crashed process; does not affect rate
// limiting (that counts by created_at regardless of status, by design).
func runReaper(ctx context.Context, log *slog.Logger, st *store.Store) {
	run := func() {
		n, err := st.ReapStaleAIRuns(ctx, reaperStaleAfter)
		if err != nil {
			log.Error("reap stale ai runs", "error", err)
			return
		}
		if n > 0 {
			log.Info("reaped stale ai runs", "count", n)
		}
	}
	run()
	ticker := time.NewTicker(reaperInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			run()
		case <-ctx.Done():
			return
		}
	}
}
