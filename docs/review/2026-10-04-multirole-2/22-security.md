# 22 · Security

**Lens:** 
**Score:** **7.5 / 10**
**Bottom line:** Waves closed the worst gaps (real lockout + dual-keyed cooldown, non-root containers, correct two-layer ideas-private RLS, clean CI) — what remains: every IP-keyed decision trusts rightmost XFF unconditionally though the Go server has its own public domain; idea author paths never re-check membership so a removed member can still convert ideas into member-visible PRDs; no security headers on the SPA; and several round-one items still open including the prod domain in tracked files.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. [Correction] SEC-10 under-severitied the rightmost-XFF defect: reliability-ops filed the same code path as REL-13 and the skeptic confirmed it High (sitewide fail-closed backstop on the documented Railway path). Treat SEC-10 and REL-13 as one remediation item; the register keeps both filed severities for the record.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 11 findings as filed, worst first.

## Findings

### SEC-10 · Medium ·  **Skeptic: CONFIRMED.**

**clientIP trusts the rightmost X-Forwarded-For entry unconditionally, but the documented deployment exposes the Go server directly (Railway public API domain; compose 0.0.0.0), so a stranger reaching the server without the appending nginx controls their IP key and defeats every IP-keyed throttle and cooldown.**

- Where: `server/internal/api/auth_throttle.go:292`
- Evidence: auth_throttle.go:287-291 states the single-nginx invariant; docs/usage/deployment.md:105-108 'server (Go, :8080 public API + private)'; docker-compose.yml:102 maps the port on all interfaces.
- Fix: Gate XFF trust on INSIDEOUT_TRUSTED_PROXIES (CIDRs; empty = RemoteAddr only), bind compose server port to 127.0.0.1, point native/CLI at the app domain. First check: canary XFF probe against the live server domain.
- Diff size: ~40 lines + 2-line compose change

### SEC-11 · Medium · 

**Idea author paths never re-check workspace membership: a removed member keeps read/edit/drop/convert rights over old ideas, and convert mints a member-visible PRD inside a workspace they were kicked out of.**

- Where: `server/internal/store/ideas.go:129`
- Evidence: ideas.go:129-131 UpdateIdea checks only authorID != actorID; :196-198 ConvertIdea same; prds_insert RLS checks only author_id = current_user_id() (row_level_security.sql:269-270).
- Fix: Add requireMember to UpdateIdea/DropIdea/ConvertIdea; tighten ideas_update/prds_insert WITH CHECK with the membership EXISTS clause; add an ex-member test case.
- Diff size: ~25 lines + migration

### SEC-12 · Medium · 

**The SPA is served with zero security headers (no CSP, nosniff, frame-ancestors, Referrer-Policy, HSTS) while the web client persists a 720-hour refresh token — one XSS is unmitigated persistent takeover.**

- Where: `client/nginx.conf:1`
- Evidence: client/nginx.conf:1-37 no add_header anywhere; refresh TTL 720h default (docker-compose.yml:88); Bearer tokens from SecureTokenStore.
- Fix: Add X-Content-Type-Options, Referrer-Policy, frame-ancestors 'none', HSTS at the edge, and a conservative Flutter-web CSP (default-src 'self'; connect-src 'self' <API origin>; relax from console violations).
- Diff size: ~10 lines

### SEC-13 · Low · 

**The NULL-context trust ('unset app.user_id means system') is still convention-enforced, and the all-RLS-tables walk test requested by round-one SEC-04 still does not exist.**

- Where: `server/db/migrations/20260720150000_row_level_security.sql:33`
- Evidence: row_level_security.sql:33-34 current_user_id() IS NULL trusted; authz_test.go has three scenario tests + ideas-only privacy test.
- Fix: DB-gated regression walking every RLS table asserting zero rows for a non-member actor and a defined result for the unset context; longer term an explicit app.system flag.

### SEC-14 · Low · 

**SEC-08 was marked redacted, but the production domain still appears in a Go source comment and five tracked docs, including the durable dogfood account address.**

- Where: `server/internal/api/github_webhook.go:26`
- Evidence: github_webhook.go:26 '// (https://<prod-domain>/api/v1/hooks/github). Signature first,'; docs/changelogs/2026-08-21-dogfood-and-cleanup.md:8 names the account at that domain; four more docs repeat the host.
- Fix: Sweep occurrences to <prod-domain> placeholders; add a grep guard to the docs guardrail script.
- Diff size: ~10 lines

### SEC-15 · Low · 

**Prior SEC-05 unfixed: Vary: Origin is still set only inside the allowed-origin branch.**

- Where: `server/internal/api/cors.go:60`
- Evidence: cors.go:60-62 the Vary set sits inside if originAllowed(...).
- Fix: Set Vary: Origin unconditionally at the top of withAllowlistCORS.
- Diff size: 1 line

### SEC-16 · Low · 

**Prior SEC-06 unfixed: INSIDEOUT_DEV_CORS=1 reflect-any-origin with credentials still boots without complaint next to a non-local PUBLIC_URL.**

- Where: `server/internal/config/config.go:93`
- Evidence: config.go:93 DevPermissiveCORS parsed with no cross-check; middleware.go:128-129 reflects caller Origin with Allow-Credentials true.
- Fix: Refuse to start when DevPermissiveCORS is set and PUBLIC_URL is neither localhost nor 127.0.0.1.
- Diff size: ~5 lines

### SEC-17 · Low · 

**The global 1 MiB withMaxBody wraps the webhook route, making the handler's 8 MiB LimitReader dead code — GitHub push payloads above 1 MiB fail 400 and the hook eventually gets disabled.**

- Where: `server/internal/api/github_webhook.go:36`
- Evidence: middleware.go:92 maxBodyBytes = 1 MiB applied around the whole mux (server.go:80-83); github_webhook.go:36 LimitReader 8<<20.
- Fix: Give the webhook route its own limit (mount outside withMaxBody with the 8 MiB reader or a per-route override).
- Diff size: ~10 lines

### SEC-18 · Low · 

**The presence/cursor 'client' id is caller-chosen and unauthenticated, so any project member can overwrite or evict another viewer's presence entry and cursor by passing their client id.**

- Where: `server/internal/api/presence_routes.go:58`
- Evidence: presence_routes.go:58-61 client := r.URL.Query().Get("client") used as the registry key; Leave removes whatever id the connector claimed.
- Fix: Namespace client ids under the authenticated user (userID + ':' + suffix) so a session can only act on its own entries.
- Diff size: ~10 lines

### SEC-19 · Info · 

**Conversion publishes the idea's title to all workspace members (PRD carries ideaId + verbatim title; prds_select is member-readable) — private-by-default ends the moment the author converts.**

- Where: `server/internal/api/prds.go:32`
- Evidence: prds.go:32 IdeaID exposed; store/ideas.go:219 insertPrd copies idea.Title; prds_select grants any member read.
- Fix: Confirm intent: if conversion is the explicit share act, document at the convert endpoint and PRODUCT.md; else make converted PRDs author-private until a share action.

### SEC-20 · Info · 

**idea_list's tool description promises the old semantics ('a workspace's idea inbox with full content') but the endpoint now returns caller-scoped rows only.**

- Where: `server/internal/mcpserver/product_tools.go:249`
- Evidence: product_tools.go:249 addTextTool("idea_list", "List a workspace's idea inbox with full content" vs store ListIdeas author filter.
- Fix: Reword to 'List the calling user's private ideas in a workspace'; sweep other pre-privacy descriptions.

