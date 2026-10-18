# 22 · Security

**Lens:** route auth coverage, JWT/refresh/cookies/CORS, rate limiting and enumeration, SSRF and injection, tenant isolation via repo link, secret handling in logs/errors, SSE auth, MCP/CLI token trust, supply chain (L9/L10/L11), Docker image and nginx, dependency/config hygiene  
**Score:** **6.8 / 10**  
**Bottom line:** Sound security design (AST-enforced route auth, RLS, HMAC webhooks, hashed rotating refresh tokens) undermined by abuse-resistance gaps: the login-code attempt budget is bypassed by a concurrent burst and the IP throttle trusts a client-controlled XFF tail off-edge, both confirmed by running. Supply chain is unsigned and tag-pinned. Nothing Critical.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| SEC-01 | High | DOWNGRADED | Medium | **[Correction]** The mechanism is real and I reproduced it on a private Postgres 18.4. The code is wrong about the size of the impact. VerifyLoginCode runs the hash-compare UPDATE first (auth_tokens.go:82). The attempts increment is a separate UPDATE that runs only after a miss (:93-98), so nothing serializes them. At the store level, with a … |

---

## Scope

Security lens on InsideOut at main @ 9f0b1c7 (clean detached worktree). Read: route table and auth wrappers in `server/internal/api`, `auth`, `store` authz and token paths, `github` (webhook, App auth, REST client), `mail`, `config`, `mcpserver`, `cmd/insideout-mcp`, `clientconfig`, `Dockerfile`, `docker-compose.yml`, `client/nginx.conf`, `.github/workflows/*`, `scripts/bootstrap.sh`. Three issues were confirmed by running scratch tests (kept outside the tree) against a private Postgres 18.4 cluster with CI-parity roles; the cluster is stopped. History and `hosts/` exposure (L7, L8) are left to the public-release reviewer. No network beyond localhost; the hosted instance (`<prod-host>`) was not contacted.

| Aspect | Score | Why |
|---|---|---|
| Authn on routes | 8.5 | AST test enforces auth wrapper on every route; public list is explicit |
| Tokens, cookies, CORS | 7.5 | HttpOnly/Lax/Secure, hashed rotating refresh; no reuse revocation, no iss/aud |
| Rate limiting, enumeration | 5.5 | Throttle is bypassable by concurrency and (off-edge) by XFF; timing/409 oracles |
| Injection, SSRF, tenancy | 7.5 | Parameterised SQL, RLS, prompt fence; repo-link and `..` segments soft |
| Secrets in logs/errors | 8.0 | Dev mail body gated; admin probe surfaces only a last error |
| Supply chain, image | 5.5 | Tag-pinned actions, unsigned same-origin artifacts, distroless image is good |
| Overall | 6.8 | Sound design, a few real defects at the abuse-resistance layer |

## What is good

- `server/internal/api/routes_require_auth_test.go` parses every `HandleFunc` and fails if a non-listed route lacks the auth wrapper. This is the strongest control in the repo.
- `server/internal/api/github_webhook.go` verifies the HMAC before processing, and `attachGuideEvidence` checks leaf ids so a delivery cannot attach evidence across tenants.
- `server/internal/agent/prompt_fence.go` fences user PRD text in `<prd_data>`; `server/internal/clientconfig/clientconfig.go` writes tokens 0600 via temp file and rename.
- `server/Dockerfile` ends in distroless, `USER nonroot`, with a self-probing healthcheck; `client/nginx.conf` ships CSP, HSTS, nosniff and frame denial.
- `server/internal/api/auth_email_flows.go` builds mail links from `cfg.PublicURL`, not the Host header, so there is no host-header poisoning. Login uses a dummy-hash timing equaliser, and reset-password revokes sessions.

## Findings

1. **High, SEC-01: login-code attempt budget is bypassed by concurrency (confirmed by running).** `server/internal/store/auth_tokens.go:82` first runs a consume-on-match `UPDATE`, and only on miss runs a second `UPDATE` at `:95` (`SET attempts = attempts + 1,`). The two are not atomic and the API throttle counts failures only after the response. In 20 trials, 60 concurrent guesses where the correct one was last were accepted in 19, despite `LoginCodeMaxAttempts = 5`. A burst turns a 5-guess budget into an unbounded one for the code lifetime.
2. **Medium, SEC-02: throttle and resend-cooldown key on a client-controlled XFF tail off-edge (confirmed by running).** `server/internal/api/auth_throttle.go:300` takes the rightmost `X-Forwarded-For` entry; `client/nginx.conf:79` passes it through; `docker-compose.yml:104` publishes the server on all interfaces. 500 guesses with a rotating XFF tail: 0 throttled; fixed-address control: 492 throttled. The code comment calls this dev-only, which holds on Railway because the edge appends, but `docs/usage/deployment.md` also lists the `server` service with its own public domain and any non-Railway self-host inherits the hole.
3. **Medium, SEC-03: account enumeration oracles.** Magic-link, forgot-password and login-code requests call `s.mail.Send` synchronously only for known users, so latency differs (reasoned from code, not measured). Register returns a distinct 409. The per-IP limits in SEC-02 weaken the mitigation.
4. **Medium, SEC-04: no session-family revocation; access JWTs not revocable.** `server/internal/store/sessions.go:71` (`// ponytail: on detected reuse we only refuse this rotation.`) refuses only the replayed rotation; a stolen refresh token that rotates first keeps the thief live. `server/internal/auth/jwt.go` is HS256 with no `iss`/`aud`, and logout cannot kill a live 15-minute access token.
5. **Medium, SEC-05: MCP and CLI token is not bound to an API base.** `server/cmd/insideout-mcp/main.go:22` reads `INSIDEOUT_API` and ignores the saved config `api_base`, while `server/internal/mcpserver/server.go:51` `sessionToken()` falls back to the saved token regardless of base. Repointing `INSIDEOUT_API` sends the hosted bearer to that host. Admin tools (`signup_mode`, `invite_revoke`) are exposed to the agent.
6. **Medium, SEC-06: repo link lets a member aim the server-wide PAT at any repo.** `server/internal/github/github.go:50` `isRepoSegment` accepts `.` and `..` (confirmed: `../..`, `./..` and `../user` parse as a repo); `fetchCommitPage` then sends `Authorization: Bearer` with `GITHUB_TOKEN` to `api.github.com/repos/<seg>/<seg>/commits`. `ParseRepoURL` also accepts `https://evil.example/x/github.com/a/b`, and `SetProjectRepo` stores the raw string. Not exploitable for SSRF (host fixed), but any project member can read commit metadata of private repos the PAT can see, and `..` rewrites the request path.
7. **Medium, SEC-07: release and supply chain.** `.github/workflows/release.yml:66` `gh release upload ... --clobber`, unsigned tarballs, `checksums.txt` from the same origin that `scripts/bootstrap.sh` verifies against (checksum proves transport, not provenance). All Actions are tag-pinned (`actions/checkout@v4`); `ci.yml` and `ubuntu-install.yml` have no `permissions:` block; `ubuntu-install.yml:63` accepts go1.25 though the floor is 1.26.0; `INSIDEOUT_RELEASE_BASE` is not forced to https.
8. **Low, SEC-08: SSE streams authorise once.** `server/internal/api/presence_routes.go:50` calls `GetProjectForMember` at connect only; no max lifetime, no per-user cap, no re-check after removal from the project.
9. **Low, SEC-09: hardening gaps.** `.env.example:68` placeholder `change_me_to_a_long_random_string` is 36 chars and passes the length-only check, so a copied file boots with a public JWT secret. `DevPermissiveCORS` is guarded only by a loopback `PublicURL`. `server/internal/api/github_webhook.go:42` `io.ReadAll` takes up to the 8 MiB cap before the HMAC check.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L9 actions tag-pinned, no permissions in ci/ubuntu-install | Confirmed | `actions/checkout@v4` at `release.yml:27`; extended with SEC-07 `--clobber` and no attestation |
| L10 unsigned same-origin checksums, `--clobber` | Confirmed, extended | `release.yml:66`; bootstrap does not enforce https on the release base |
| L11 go1.25 vs 1.26 floor | Confirmed | Toolchain mismatch means CI can pass on a compiler the image never uses |
| L15 `.env.example` and compose passwords | Confirmed, extended | Compose db is loopback-only, but `.env.example:68` JWT placeholder passes validation |
| L16 image job and Dockerfile | Confirmed | Distroless, nonroot, healthcheck |
| L5 hosted headers sound | Not re-probed | No network allowed |
| L1 time bomb | Not filed | Coordinator-owned; sibling wall-clock tests not in the security lens |

## Recommendations

1. Make the login-code check one atomic statement: increment attempts in the same `UPDATE` that compares the hash, or take `SELECT ... FOR UPDATE` on the live row.
2. Count throttle failures before verification (reserve a slot, refund on success) so bursts cannot outrun the counter.
3. Add `TRUSTED_PROXY_HOPS` to config and take the Nth-from-right XFF entry; default to `RemoteAddr` when unset. Bind compose `SERVER_PORT` to `127.0.0.1`.
4. Reject `.`/`..` in `isRepoSegment`, store the canonical `owner/repo`, and scope `GITHUB_TOKEN` to public-read or drop it in favour of installation tokens.
5. Add `family_id` to sessions and revoke the family on refresh reuse; add `iss`/`aud` to the JWT.
6. Bind the CLI/MCP token to its `api_base` and refuse a mismatched `INSIDEOUT_API`; gate admin MCP tools behind an explicit flag.
7. Sign releases (cosign or attestation), publish checksums out of band, drop `--clobber`, SHA-pin Actions, add `permissions: contents: read` to ci and ubuntu-install, enforce https in bootstrap.
8. Send mail asynchronously (queue) and return identical timing; reject placeholder JWT secrets by denylist at startup.

## What this role would do first

Fix SEC-01 today: it is about twenty lines, it affects the hosted instance, and it is the only finding that turns a rate limit into nothing without any deployment caveat. Then close SEC-02 and SEC-06 together.

Rotate or harden today: set a real random `INSIDEOUT_JWT_SECRET` in every environment that ever copied `.env.example`; scope or remove the server-wide `GITHUB_TOKEN`; bind the compose server port to loopback; keep `MAIL_DEV_BODY` and `DEV_CORS` off outside local runs. Do not expose: the Railway `server` service on its own public domain (put only the nginx app on the edge), the compose server port, Postgres, or the admin MCP tools to untrusted agents.
