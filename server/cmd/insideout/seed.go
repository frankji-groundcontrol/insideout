package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

const (
	seedEmail    = "demo@insideout.local"
	seedPassword = "demo12345"
	seedUsername = "Demo"
)

func runSeed(ctx context.Context, log *slog.Logger, st *store.Store) {
	if _, err := seedDemo(ctx, log, st, seedEmail); err != nil {
		log.Error("seed", "error", err)
		os.Exit(1)
	}
}

// seedDemo creates a demo user, workspace, project, idea, and PRD for local
// development — implemented in Go (not raw SQL) so the demo user's
// password is hashed through the exact same argon2id path the server uses
// at login time. See docs/plans/2026-07-20-go-rewrite/01-database.md §3.
// It returns errors rather than exiting so tests can run it against a
// throwaway user.
func seedDemo(ctx context.Context, log *slog.Logger, st *store.Store, email string) (seeded bool, err error) {
	hash, err := auth.HashPassword(seedPassword)
	if err != nil {
		return false, fmt.Errorf("hash password: %w", err)
	}

	user, err := st.CreateUser(ctx, email, hash, seedUsername)
	if errors.Is(err, store.ErrConflict) {
		user, err = st.GetUserByEmail(ctx, email)
	}
	if err != nil {
		return false, fmt.Errorf("create/get demo user: %w", err)
	}

	ws, err := st.CreateWorkspace(ctx, user.ID, "InsideOut Demo Team", "Seed workspace for local development")
	if err != nil {
		return false, fmt.Errorf("create workspace: %w", err)
	}

	project, err := st.CreateProject(ctx, user.ID, ws.ID, "Onboarding Revamp", "Redesign the first-run experience")
	if err != nil {
		return false, fmt.Errorf("create project: %w", err)
	}
	if _, err := st.AddProjectUpdate(ctx, user.ID, project.ID, "progress", "Kicked off discovery interviews with 5 users."); err != nil {
		return false, fmt.Errorf("add project update: %w", err)
	}

	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "In-app changelog widget", "Users keep asking what changed after each release. A small bell icon with a changelog popover could fix this.")
	if err != nil {
		return false, fmt.Errorf("create idea: %w", err)
	}

	prd, conv, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		return false, fmt.Errorf("convert idea: %w", err)
	}

	// The demo login is a public dev constant, documented in
	// docs/usage/local-development.md, so logging it is deliberate. That is
	// also why seed is dev-only: never point it at a shared or production
	// database.
	log.Info("seed: done",
		"email", email, "password", seedPassword,
		"workspace_code", ws.Code, "project_id", project.ID.String(),
		"idea_id", idea.ID.String(), "prd_id", prd.ID.String(), "conversation_id", conv.ID.String(),
	)
	return true, nil
}
