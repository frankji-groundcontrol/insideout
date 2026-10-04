# 10 · Architect

**Lens:** 
**Score:** **7.6 / 10**
**Bottom line:** Post-remediation tree architecturally sound and honest; risks are drift and parity, not live defects; CI/ship toolchain split should be fixed before trusting green builds as deployable.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. [Correction] ARCH-01 was downgraded High→Medium by the skeptic: all three pin sites are real, but nothing on the documented path fails today — CI certifies a different toolchain than the artifact build, which is divergence risk, not present wrong behavior.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 10 findings as filed, worst first.

## Findings

### ARCH-01 · High ·  **Skeptic: DOWNGRADED → Medium.**

**CI builds/tests the client on Flutter 3.47.0 while the shipped web image builds on 3.44.0 — the toolchain CI certifies is not the toolchain that produces the production artifact.**

- Where: `.github/workflows/ci.yml:81`
- Evidence: .github/workflows/ci.yml:81 'flutter-version: 3.47.0' vs client/Dockerfile:3 'FROM ghcr.io/cirruslabs/flutter:3.44.0 AS build' (docs/architecture/deployment.md:23 still documents 3.44.0).
- Fix: Pin CI to the Dockerfile version (one line), or bump client/Dockerfile to 3.47.0 and update deployment.md in the same change; keep both pins adjacent so they move together.
- Diff size: ~2-6 lines

### ARCH-02 · Medium · 

**The documented live-server SSE exercise (AGENTS.md non-negotiable, smoke.sh S1) cannot run in CI: it hard-sources a repo-root .env that does not exist on a runner.**

- Where: `server/scripts/smoke.sh:26`
- Evidence: smoke.sh:26 'set -a; source "$REPO_ROOT/.env"; set +a' under 'set -euo pipefail'; CI has no smoke step; conversations_test.go is httptest, not a booted server.
- Fix: Make smoke.sh fall back to ambient env when .env is absent, then add a CI step running it against the job's postgres.
- Diff size: ~10-20 lines

### ARCH-03 · Medium · 

**Migrate-on-boot plus process-local throttle/cooldown/presence embed an undocumented single-instance invariant; concurrent replicas race on migrations and split rate-limit state.**

- Where: `server/cmd/insideout/main.go:114`
- Evidence: main.go:114 'if _, err := st.Migrate(ctx); err != nil {' before listening; migrate.go:95 plain INSERT into schema_migrations — two replicas race, loser's whole tx rolls back and it exits non-zero. Only presence is documented single-instance (product-subsystems.md:57-58).
- Fix: Session-level pg_advisory_lock around Migrate's apply loop; document the process-local state inventory (throttle, cooldown, presence, AI gate) in deployment.md.
- Diff size: ~15 lines + docs

### ARCH-04 · Medium · 

**The author-or-admin idea-privacy rule now exists in three hand-maintained renderings (RLS via _is_admin, GetIdeaForMember's inline EXISTS, DropIdea/UpdateIdea Go checks); drift fails safe but silent.**

- Where: `server/internal/store/ideas.go:102`
- Evidence: ideas.go:102-104 inline EXISTS admin check vs 20261004120000_ideas_private_by_default.sql:11-12 'author_id = insideout.current_user_id() OR insideout._is_admin(...)'.
- Fix: Drop the explicit predicate in GetIdeaForMember and let ideas_select filter inside the already-user-scoped tx, or call the same _is_admin helper; add cross-reference comments naming the paired source of truth.
- Diff size: ~10 lines

### ARCH-05 · Medium · 

**client-IP extraction (rightmost XFF) implemented twice with mirrored comments; the single-appending-proxy assumption is undocumented at the deployment level.**

- Where: `server/internal/api/auth_throttle.go:293`
- Evidence: auth_throttle.go:293-295 rightmost-XFF split duplicated verbatim at auth_cooldown.go:71-73 as cooldownClientIP ('mirrors the throttle middleware's IP extraction on purpose').
- Fix: Extract one shared clientIP(r) used by both middleware; pin the one-appending-proxy invariant in its comment and deployment.md.
- Diff size: ~15 lines

### ARCH-06 · Low · 

**quietAfter (14d) is a product-visible board policy hardcoded in the API layer with no env/config surface; placement decision unrecorded.**

- Where: `server/internal/api/projects.go:43`
- Evidence: projects.go:43 'const quietAfter = 14 * 24 * time.Hour' computed per row in projectResponse; absent from the env catalog the scripts suite tests.
- Fix: Decide and record: keep as product const named in PRODUCT.md, or promote to INSIDEOUT_QUIET_AFTER in config with the 14d default + env catalog entry.
- Diff size: ~8 lines

### ARCH-07 · Low · 

**CI models only the shared-instance privilege shape; the dedicated-instance bootstrap (app owns DB, migrate without DATABASE_OWNER_URL) is never exercised.**

- Where: `.github/workflows/ci.yml:50`
- Evidence: ci.yml:50 'ALTER DATABASE postgres OWNER TO insideout_owner;' + always setting DATABASE_OWNER_URL vs migrate.go's comment naming the dedicated-instance first-run path; compose provisions app as DB owner.
- Fix: Add a CI step that grants the database to insideout_app and re-runs 'insideout migrate' with DATABASE_OWNER_URL unset.
- Diff size: ~10 lines

### ARCH-08 · Low · 

**List-endpoint membership enforcement lives in handlers as a separate precheck transaction rather than inside the store's withUserContext, making the store rule caller-dependent.**

- Where: `server/internal/api/ideas.go:52`
- Evidence: ideas.go:52 GetMembership precheck before ListIdeas, whose doc disclaims ('Requires the caller to have already verified membership.'); same in projects.go:85; contrast CreateIdea's in-tx requireMember.
- Fix: Move requireMember inside ListIdeas/ListProjectsForWorkspace's withUserContext, returning ErrNotFound for non-members; drop the handler precheck.
- Diff size: ~20 lines

### ARCH-09 · Low · 

**The send cooldown is package-level global state while its sibling throttle hangs off Server — two lifecycles for one mechanism pair.**

- Where: `server/internal/api/auth_cooldown.go:21`
- Evidence: auth_cooldown.go:21 'var authSendCooldown = newSendCooldown(authResendCooldown)' vs server.go:47 authThrottle as a Server field.
- Fix: Standardize both on Server-scoped state constructed in NewServer.
- Diff size: ~10 lines

### ARCH-10 · Low · 

**migrateQueryRow bypasses beginMigrateTx's role guard — an open extension point for future pre-bootstrap reads running as the wrong role.**

- Where: `server/internal/store/migrate_role.go:64`
- Evidence: migrate_role.go:63-65 'func (s *Store) migrateQueryRow(...) pgx.Row { return s.migrator().QueryRow(...) }' — no prepareMigrateConn unlike migrateExec/beginMigrateTx.
- Fix: Route it through a guarded transaction or rename migrateBootstrapQueryRow with a must-stay-read-only comment.
- Diff size: ~8 lines

