// Command insideout runs the InsideOut API server. Subcommands:
//
//	insideout            run the HTTP server
//	insideout migrate     apply pending SQL migrations and exit
//	insideout seed        create demo data for local development
//	insideout health      container HEALTHCHECK: probe /healthz over loopback
//	insideout promote-admin <email>  grant admin (owner pool); creates the account, password on stdin, when absent
//	insideout help        list every command (also -h, --help)
//	insideout version     print the build version (also --version)
//
// The product CLI verbs (login, ideas, prd, roadmap, setup, …) dispatch
// from runClientCommand in cli.go before any server env or DB is touched;
// the cliVerbs table there is the verb inventory `help` prints — verbs
// enumerated here went stale (review-3 ARCH-04), so this comment
// deliberately does not list them. An unknown verb exits 2 (help.go).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
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

	// Product CLI commands need no server env or DB — dispatch them
	// before any config/store setup (docs/plans/2026-08-21-cli-mcp-parity.md).
	if handled, code := runInterruptible(os.Args[1:]); handled {
		os.Exit(code)
	}
	// help, version and a mistyped verb answer here too: none of them may
	// reach config.Load() and its "DATABASE_URL is required" (rv4-USER-02).
	if handled, code := runFrontDoor(os.Args[1:], os.Stdout, os.Stderr); handled {
		os.Exit(code)
	}

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
	if cfg.DatabaseOwnerURL != "" {
		if err := st.AttachMigratePool(ctx, cfg.DatabaseOwnerURL); err != nil {
			log.Error("open migrate store", "error", err)
			os.Exit(1)
		}
	}

	switch cmd := args(); cmd {
	case "migrate":
		runMigrate(ctx, log, st)
	case "seed":
		runSeed(ctx, log, st)
	case "promote-admin":
		if len(os.Args) != 3 {
			log.Error("usage: insideout promote-admin <email>  (a new account's password is read from stdin, never argv)")
			os.Exit(2)
		}
		runPromoteAdmin(ctx, log, st, os.Args[2])
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

// shutdownDrain is how long in-flight requests (coach turns) get to finish
// once shutdown starts, before their contexts are cancelled. The coach bounds
// its terminal writes at 10s (agent persistTimeout), so 20s + 10s fills the
// 30s Shutdown window. A var only so the shutdown test can shorten it.
var shutdownDrain = 20 * time.Second

func runServe(ctx context.Context, log *slog.Logger, st *store.Store, cfg *config.Config) {
	// Before anything runs on the pool, including the migrations: a role that
	// skips row-level security must not serve (rv4-DATA-04). migrate and
	// promote-admin never reach here and keep using the owner role.
	if err := st.CheckServeRole(ctx); err != nil {
		log.Error("refusing to serve", "error", err)
		os.Exit(1)
	}
	if _, err := st.Migrate(ctx); err != nil {
		log.Error("migrate on startup", "error", err)
		os.Exit(1)
	}

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTTL)

	var streamer agent.ChatStreamer
	var planner agent.RoadmapPlanner
	if cfg.AIAuthToken == "" {
		log.Info("INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach")
		streamer = agent.NewTemplateStreamer()
		planner = agent.NewTemplateRoadmapPlanner()
	} else {
		s, err := agent.NewLLMStreamer(cfg.AIBaseURL, cfg.AIAuthToken, cfg.AIModel, cfg.AISchema)
		if err != nil {
			log.Error("create llm streamer", "error", err)
			os.Exit(1)
		}
		streamer = s
		planner = agent.NewAnthropicRoadmapPlanner(s)
		if checker, ok := s.(interface {
			CheckModel(context.Context) (bool, []string, error)
		}); ok {
			checkModelAtStartup(ctx, log, checker, cfg.AIModel)
		}
	}
	coach := agent.New(st, streamer, log)

	srv := api.NewServer(st, tokens, cfg, log, coach, planner)

	// Every request context derives from baseCtx, so cancelling it after the
	// drain sends an in-flight coach turn down its context.Canceled path: the
	// streamed text is kept and the run is marked failed, instead of both
	// dying with the process.
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		// ReadTimeout bounds a slow-drip body (pairs with the body caps in
		// api/middleware.go: 1 MiB everywhere, 8 MiB on the GitHub webhook
		// route); IdleTimeout reaps keep-alive connections. No
		// WriteTimeout: it would cap the whole response and cut the SSE
		// streams, which arm a deadline per write instead
		// (httpx.ArmWriteDeadline).
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
		BaseContext: func(net.Listener) context.Context { return baseCtx },
	}
	// Shutdown never cancels handler contexts: end the passive streams at
	// once (an open canvas or progress tab would hold the whole window), and
	// cut whatever is still in flight after the drain.
	httpServer.RegisterOnShutdown(func() {
		srv.CloseStreams()
		time.AfterFunc(shutdownDrain, cancelBase)
	})

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
type modelChecker interface {
	CheckModel(context.Context) (bool, []string, error)
}

func checkModelAtStartup(ctx context.Context, log *slog.Logger, s modelChecker, model string) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ok, available, err := s.CheckModel(checkCtx)
	if err != nil {
		log.Warn("could not verify INSIDEOUT_LLM_MODEL against the provider's /models — proceeding anyway", "model", model, "error", err)
		return
	}
	if !ok {
		log.Warn("INSIDEOUT_LLM_MODEL is not in the provider's model list — the coach will fail at the first request", "model", model, "available", available)
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
