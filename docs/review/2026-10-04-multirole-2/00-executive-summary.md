# Executive summary — multirole review 2

| | |
|---|---|
| **Repository** | insideout (private) |
| **Branch / SHA** | main @ 26db979 |
| **Date** | 2026-10-04 |
| **Cross-role mean** | **7.3 / 10** (range 6.5–8.4) |
| **Register** | 108 unique: 0 Critical · 7 High · 53 Medium · 46 Low |

**Verdict.** The tree that closed the first review's five waves is visibly
healthier: no Criticals, the CI spine exists, the new tests are the strongest in
the repo, and the line-budget rule holds everywhere it applies. The seven
confirmed Highs are narrow and fixable — one agent bug that production never
surfaces (the critic's empty `messages` array), one deployment-topology defect
(per-IP throttle collapsing to the Railway edge IP), and a five-item web
failure-path cluster (Drop, SSE 401, raw errors, popup-blocked export, proposal
loop). The large Medium band is dominated by exactly what the owner asked for:
an honest PRODUCT.md-versus-code ledger of features left to implement, kept
Medium by the skeptics because PRODUCT.md's own header frames them as end-state.

## Score table

| Reviewer | Score | One-line reason |
|---|---|---|
| code-quality | 8.4 | Rule holds; residue is one widget duplication and helper hygiene |
| architect | 7.6 | Sound structure and CI spine; toolchain pin divergence is the wobble |
| feature-implementation | 7.5 | 50 MCP tools verified; Branches/Merge/Decision-Log ledger honest |
| security | 7.5 | No secrets, auth tight; XFF severity understated (see critic) |
| reliability-ops | 7.5 | Limits and retries mostly present; XFF collapse is the outlier |
| fullstack | 7.2 | Happy paths solid; failure paths (popup, 401, errors) are the debt |
| testing | 7.0 | Real DB-gated suite, 0 skips; guard scope, no `-race`, no smoke |
| ai-developer | 6.8 | Good trust boundaries; the critic bug is a silent prod failure |
| project-manager | 6.5 | Ledger rich; deploy-record hygiene and human gates open |
| user | 6.5 | The confirmed Highs live here — the UI breaks where users actually go |

## Top ten by user impact

| # | Finding | Where | Sev | Fix size |
|---|---|---|---|---|
| 1 | AIDEV-09 critic sends empty `messages` → every prod critic call 400s, silently skips | `server/internal/agent/critic.go:231` | High | ~20 lines + wire test |
| 2 | REL-13/SEC-10 per-IP throttle collapses to the edge IP — sitewide fail-closed lockout | `server/internal/api/auth_throttle.go:292` | High | canary + ~15 lines + two-hop test |
| 3 | USER-10 coach SSE never refreshes on 401 — permanent "stream failed" loop | `client/lib/api/sse_transport_web.dart:23` | High | ~40 lines |
| 4 | USER-09 Drop never visibly drops; dead confirm string; raw 'dropped' status | `client/lib/features/workspace/ideas_page.dart:107` | High | ~30 lines |
| 5 | USER-11 raw English server errors rendered in a zh-CN UI (12+ sites) | `client/lib/features/workspace/workspace_page.dart:86` | High | ~60 lines |
| 6 | FS-01 popup-blocked print export silently no-ops (dead fallback) | `client/lib/api/download_web.dart:19` | High | 2 lines + backstop |
| 7 | FEAT-03 agent proposals have no web accept/reject loop | `client/lib/features/project/project_page.dart:125` | High | ~60 lines |
| 8 | ARCH-01 CI Flutter 3.47.0 vs Docker 3.44.0 — certifies the wrong toolchain | `.github/workflows/ci.yml:81` | Med | 1 line |
| 9 | TEST-09 skip-guard only watches `./internal/store/...` | `.github/workflows/ci.yml:64` | Med | 1 line |
| 10 | PM-13 no commit+date deploy record in the tree | `docs/changelogs/2026-10-04-remediation-wave-3-4.md:124` | Med | docs only |

## What each role would say at the door

- **Architect:** "Pin CI to the Dockerfile's Flutter version — one line, today."
- **AI Developer:** "Give the critic a one-message payload like the planner before trusting any critic pass."
- **Fullstack:** "Capture `window.open`'s null return; your export fallback is dead code."
- **Project Manager:** "The deploy happened — write it down where a stranger can verify it."
- **User:** "I dropped an idea and it's still on my wall, in English."
- **Feature Implementation:** "Stop by the proposals: they're stuck as notes with no Accept button."
- **Code Quality:** "Extract the shared band scaffold before the two roadmap views drift."
- **Security:** "Decide who you trust on X-Forwarded-For before the throttle matters."
- **Testing:** "Widen the skip-guard to the whole suite and add `-race` to CI."
- **Reliability & Ops:** "There is no backup story; decide one before the next migration."

## Effort table

| Bucket | Items | Estimate |
|---|---|---|
| One-liners, one commit | FS-01, ARCH-01, TEST-09, CQ-02..05, CC-02, PM-13 record | ~0.5 day |
| Web failure-path cluster | USER-09, USER-10, USER-11 | 1–2 days |
| Critic wire bug | AIDEV-09 (+ wire-level test, live exercise) | 0.5–1 day |
| Proxy/throttle depth | REL-13/SEC-10 (canary curl, config, two-hop test) | 1 day |
| Proposal web loop | FEAT-03 | 1 day |
| Mechanical cleanups | CQ-01 band scaffold, CQ-03/04 date helper | 0.5 day |
| Backup + deployment-event decisions | CC-03, CC-01 | 0.5–1 day |
| First feature slices | FEAT-02 decisionNote UI, FEAT-01/PM-11 Branch slice 1, PM-12 roles | 1–2 days per slice |
| **Total to clear waves 1–3** | | **~5–7 focused days** |

## What not to do yet

- **No Product Branches/Merge or per-project authority work** until the wave-1/2
  fixes land — PM-12 built on the superseded-assumption confusion (CC-02) would
  bake in the wrong model.
- **No Flutter pin bump** (either direction) as part of another change — do the
  one-line CI pin first, then a deliberate upgrade with its own build check.
- **No pagination/load-more infra** until the API pagination shape is designed;
  the current load-more Mediums are client-side presentation only.
- **No splitting of `en.dart`/string tables or `roadmap_test.go`** — the critic
  and code-quality both judge them inside the rule's intent; record the
  exemption once instead.
- **No new product surface in the admin/invite area** — it reviewed clean twice;
  spend the attention where the confirmed Highs are.
