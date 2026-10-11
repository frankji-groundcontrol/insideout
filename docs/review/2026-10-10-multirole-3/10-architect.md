# 10 · Architect

**Score:** **7.5 / 10** · 5 findings (1 High+)

**Bottom line:** The layering survived the 90-commit wave honestly — api never imports agent, CLI and MCP both ride internal/apiclient over the one /api/v1 contract, and the shell/Go provisioning twins are mechanically test-pinned — but the wave left one hard break: server/go.mod now demands Go 1.26.0 while server/Dockerfile still builds on golang:1.25-alpine, so every documented image-build path (compose self-host and the Railway server service) fails at `go mod download`, and no CI lane builds the server image to catch it. Around that: the stated 'Go 1.25+' minimum is stale in five install-front-door files, the email funnel has zero delivery signal and has never met a real provider, and the backend architecture doc's layout tree no longer describes the tree (20 internal packages, a second binary, a 33-file CLI). The PRODUCT.md gap set I re-verified (Product Branch/Merge, structured Decision Log, facts layer, personas, lenses, feedback invitations, dependencies) is exactly the already-dispositioned review-2 register — no regressions, no re-filing. The structural decision facing the owner is not another build slice: it is to cut the first v* release and deploy the verified wave, which forces the toolchain fix, the four pending migrations, and the first live LLM/mail exercise.

---

## Architect review — review 3, main @ 754c502

**Scope and method.** I read the routers (README.md, CLAUDE.md/AGENTS.md contract, docs/index.md), the architecture set (backend.md, database-and-rls.md skimmed via index claims, PRODUCT.md), both server entry points (server/cmd/insideout/main.go, server/internal/api/server.go), the mcpserver/apiclient/store/agent layering, config, packaging (server/go.mod, client/pubspec.yaml, both Dockerfiles, docker-compose.yml), the setup skill, and the disposition ledger's coverage of prior architect rows. Commands I ran this session: `go list -deps ./internal/api ./internal/agent` (import graph), `go list -m -json all | grep GoVersion | sort | uniq -c` (dependency floor), greps counting the MCP tool registry (52), `gh release list` (empty), and the git history of go.mod/Dockerfile. **Not run:** `docker compose build server` — no docker binary in this sandbox; the build-break finding rests on the pinned files plus the in-repo BUG-005 record of the identical failure signature. Prior reviews 1–2 and frontend-cobalt are fully dispositioned per the ledger; I re-file nothing from them.

**Verdict: 7.5/10.** The architecture is honest where it claims to be, and unusually disciplined at the seams that usually rot. What caps the score is one hard break on the documented deploy path, an install front door that states the wrong toolchain minimum, and a real-user funnel (email) that has never met its provider and shows no signal when it fails.

### Findings

| ID | Sev | Where | Finding | Fix |
| --- | --- | --- | --- | --- |
| ARCH-01 | High | server/Dockerfile:1 vs server/go.mod:3 | go.mod demands `go 1.26.0`; the image builds on `golang:1.25-alpine` and fails at `RUN go mod download` under GOTOOLCHAIN=local — the exact BUG-005 symptom (issues/2026-07-20-bug-005:5). Documented paths hit it: deployment.md:27 `docker compose up -d server client`, deployment.md:169 (Railway builds this Dockerfile). No CI lane builds the server image, so nothing catches it; go.mod moved to 1.26.0 on 10-07 (9493e45) after the Dockerfile pinned 1.25 (417e45a, 10-04). | No dependency needs >1.25.0 (`go list -m -json all`: max 1.25.0) — lower the directive via tidy, or bump the base image; add a server-image CI lane mirroring ci.yml:141-143's client principle. *Corrected 2026-10-11: the dependency-floor read is wrong — `x/sys` and `x/term` declare `go 1.26.0` (re-verified), so lowering was not viable; the Dockerfile moved to `golang:1.26-alpine` and go.mod must not be lowered (see the [README reproduction](README.md#reproduced-by-the-coordinator)).* |
| ARCH-02 | Low | README.md:58 + 4 more | 'Go 1.25+' stated in README.md:58, installation.md:33, backend.md:3, install.sh:22; the skill asserts 'server/go.mod says `go 1.25.0`' (modules.md:12). | Sync all five after ARCH-01; make modules.md read go.mod instead of quoting it. |
| ARCH-03 | Medium | server/internal/api/auth_email_flows.go:58 | Mail-send errors are swallowed into the uniform 200 (privacy-correct, operationally blind); `mailEnabled` means only 'key set' (auth.go:46 + mail.go:45-47); healthz pings only the DB (healthz.go:10); default From is Resend's sandbox `onboarding@resend.dev` (config.go:83), which delivers only to the account owner. Lead 15 confirms these flows never ran against a real provider. | Admin test-send probe + surface last Send error; document the sandbox-sender limit; exercise one real invitee before opening the funnel. |
| ARCH-04 | Medium | docs/architecture/backend.md:12 | The 'Layout' tree describes cmd/insideout as 'main.go (serve/migrate subcommands), seed.go' and seven internal packages — reality is a 33-file CLI+setup-TUI cmd, a second binary (cmd/insideout-mcp), and 20 internal packages; main.go:7-8's doc comment still enumerates five CLI verbs of ~35. | Refresh the tree (one line per package) and point main.go's comment at cli.go's dispatch. |
| ARCH-05 | Low | README.md:35 | 'the same surface as 50 tools' (also cli-mcp-and-guidance.md:89 '(50 today)') vs 52 registered this session (37+4 product, 9 auth, 2 admin; `presence`/`progress` added in the wave). | Update both, or de-numberize and point at tools/list. |

### The layering after the 90-commit wave — checks that passed

- **No api→agent seam re-merged.** api keeps its own narrow `Coach` interface precisely so it needn't import agent (server.go:19-21); `go list -deps` confirms both directions clean. Store remains the single bottom layer.
- **No CLI/MCP drift into a parallel stack.** Both surfaces call `internal/apiclient` and print raw API JSON — cli.go:3-7 and product_tools.go:3-5 each state the anti-drift rule, and the MCP tools are declared 1:1 with CLI verbs. The email-login asymmetry I looked for is covered: `insideout login --code|--magic` (cli_auth.go:23) matches the MCP `auth_login_code_*`/`auth_magic_link_*` tools.
- **The deliberate twins are pinned, not drifting.** `docker/postgres-init/001-create-app-role.sh` and the Go `insideout setup` role provisioning end at the same state, and cli_setup_roles_test.go:57 *reads the shell script* to enforce it; the Go env writer is parity-tested against env_catalog.py via testdata/pyreader.py. This is the right pattern for every future second implementation of a contract.
- **The ~350-line splits created file-level, not package-level, coupling.** The roadmap surface (5 files, 32–317 lines) and the coach split share only their package's Server/Coach types; boundaries are unchanged, and lead 3's no-file-over-350 result plus the green build/vet confirm the splits are cosmetic-safe.
- **Defaults degrade honestly.** No LLM key → offline template coach (main.go:132-135); no mail key → logged-not-sent (mail.go:53-59); config cross-validates the sharp combinations (mail⇒PUBLIC_URL, DEV_CORS⇒loopback, config.go:108-115).
- **Fresh non-editable install:** the bootstrap one-liner 404s today — `gh release list` is empty — but README phrases it conditionally ('Once a `v*` release is tagged'), release.yml is ready, and the skill + install.sh source paths work modulo ARCH-01/02's toolchain floor.

### The two leads I was handed

**Lead 14 (PRODUCT.md vs implementation): run.** I re-verified at 754c502 that the unimplemented target set is unchanged and every member is already a dispositioned review-2 row — Product Branch/Merge (grep `product_branch|ProductBranch`: zero non-test hits; review-2 FEAT-01/PM-11), Decision Log as a structured record (only `strOpt("note", "decision log note")`, product_tools.go:123; FEAT-02/PM-10), the facts trust layer (`CandidateFact|candidate_fact`: zero; PM-17), persona grounding labels (zero; PM-14), roadmap lenses beyond agent-context (`lens` matches only audienceview; FEAT-08/PM-16), scoped external feedback invitations (no routes; PM-17), and roadmap dependencies/rebaseline (`depends_on` in store: zero). Per-Project roles are self-disclosed in PRODUCT.md's own status section (:427-431). No regressions, so per instructions I re-file none — the gap set is tracked, and the keystone slice is Product Branch fork-from-Commit, from which Merge, the Decision Log, and persona/facts layers all hang.

**Lead 15 (real-user funnel): run, folded into ARCH-03.** The code paths exist and are uniform-200 correct, but every signal that would tell an operator delivery works is absent, the default sender cannot reach a third party, and the integration has never executed against Resend.

### The one structural decision

**Make the next milestone a deploy milestone, not a build milestone.** The tree is verified to a degree most repos never reach (814 tests, race-clean, pinned surfaces) yet is entirely unshipped: nothing since 10-08 is deployed, the hosted stack is four migrations behind, the first `v*` release is untagged so the one-command install 404s, and the two integrations that define the product — the LLM coach and the email funnel — have never run against their providers (both owner-key-blocked). Deploying forces exactly the queue this review found: fix the toolchain floor (ARCH-01/02), apply the migrations, exercise mail (ARCH-03) and one live coach turn, and tag the release. The alternative — another PRODUCT.md slice on an undeployed foundation — grows the distance between 'green in CI' and 'true in production', which is already this repo's largest architectural risk. Ship the wave; then build slice one of the target.

**Next first:** build the server image once (`docker compose build server`) to turn ARCH-01 from inference into reproduction, then converge go.mod and Dockerfile on whichever floor that build proves.

## What this role would do first

Resolve the Go floor: from server/, the check that settles it is building the image path the docs promise — `docker compose build server` (or `docker build server/`) — after choosing either lowering go.mod's directive back to the dependency-computed 1.25.0 or bumping server/Dockerfile:1 to golang:1.26-alpine; docker was unavailable in this review sandbox, so this is the one reproduction still owed.

*Corrected 2026-10-11: settled by the coordinator's reproduction — `x/sys` and
`x/term` require `go 1.26.0`, so the "dependency-computed 1.25.0" option (also
offered in the "Next first" paragraph above) was never viable;
`server/Dockerfile` now builds on `golang:1.26-alpine` — do not lower the go
directive ([README reproduction](README.md#reproduced-by-the-coordinator)).*
