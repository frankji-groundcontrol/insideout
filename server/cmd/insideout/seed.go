package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// runSeed creates a demo user, workspace, project, idea, and PRD for local
// development — implemented in Go (not raw SQL) so the demo user's
// password is hashed through the exact same argon2id path the server uses
// at login time. See docs/plans/2026-07-20-go-rewrite/01-database.md §3.
const (
	seedEmail    = "demo@insideout.local"
	seedPassword = "demo12345"
	seedUsername = "Demo"
)

func runSeed(ctx context.Context, log *slog.Logger, st *store.Store) {
	hash, err := auth.HashPassword(seedPassword)
	if err != nil {
		log.Error("seed: hash password", "error", err)
		os.Exit(1)
	}

	user, err := st.CreateUser(ctx, seedEmail, hash, seedUsername)
	if errors.Is(err, store.ErrConflict) {
		user, err = st.GetUserByEmail(ctx, seedEmail)
	}
	if err != nil {
		log.Error("seed: create/get demo user", "error", err)
		os.Exit(1)
	}

	ws, err := st.CreateWorkspace(ctx, user.ID, "InsideOut Demo Team", "Seed workspace for local development")
	if err != nil {
		log.Error("seed: create workspace", "error", err)
		os.Exit(1)
	}

	project, err := st.CreateProject(ctx, user.ID, ws.ID, "Onboarding Revamp", "Redesign the first-run experience")
	if err != nil {
		log.Error("seed: create project", "error", err)
		os.Exit(1)
	}
	if _, err := st.AddProjectUpdate(ctx, user.ID, project.ID, "progress", "Kicked off discovery interviews with 5 users."); err != nil {
		log.Error("seed: add project update", "error", err)
		os.Exit(1)
	}

	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "In-app changelog widget", "Users keep asking what changed after each release. A small bell icon with a changelog popover could fix this.")
	if err != nil {
		log.Error("seed: create idea", "error", err)
		os.Exit(1)
	}

	prd, conv, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		log.Error("seed: convert idea", "error", err)
		os.Exit(1)
	}

	log.Info("seed: done",
		"email", seedEmail, "password", seedPassword,
		"workspace_code", ws.Code, "project_id", project.ID.String(),
		"idea_id", idea.ID.String(), "prd_id", prd.ID.String(), "conversation_id", conv.ID.String(),
	)
}
