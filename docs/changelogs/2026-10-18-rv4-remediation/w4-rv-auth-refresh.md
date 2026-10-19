# Wave 4, refresh grace and the cookie jar — review-4 remediation

Branch `rv4/w4-rv-auth-refresh`, base `582a4bd` (the wave-3 integration head),
commits `bd0d020` (red), `f32b6f7` (two guard tests, green on the base) and
`7f0a641` (fix), then this fragment. Findings are cited as `rv4-<ID>` because
review IDs restart per review. All four come from the whole-change review of
`d53035a..rv4/integration`, and all four are one defect, the "Browser tabs are
unchanged" ceiling that [w3-auth-network.md](w3-auth-network.md) recorded: the
10 s refresh-reuse grace (rv4-SEC-04) keeps the loser's session family alive on
the server, but the request that loses a refresh race was still answered with
deletions of both auth cookies, and a browser applies `Set-Cookie` in the order
the responses arrive, so the loser's deletions land after the winner's pair and
erase it. No migration, no CI-config row, no edit to the apiclient or the
Flutter client. No address, host, secret or token is spelled here; every
account in the runs below was a throwaway on a private database.

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-FR-F100 (Medium) | `FIXED` | `bd0d020` | `7f0a641` | a loser inside the grace gets a bare 401; the cookies and the lineage stay the winner's. INTEGRATOR TODO: `server/internal/api/coverage_session_test.go`: `TestRefreshTakesItsTokenFromTheCookieOrTheBody` pins the old cookie clearing for a replay inside the grace (line 147); swap `expectCleared` for a "nothing set or deleted" check, the patch is below and a gate needs it. INTEGRATOR TODO: `docs/architecture/backend.md`: line 84, after "is only a lost race (rv4-SEC-04)", say the replay is refused with a 401 that sets and deletes no cookie. |
| rv4-FR-F108 (Medium) | `FIXED` | `bd0d020` | `7f0a641` | same defect on both loser branches (lookup miss inside the grace, `RotateSession` conflict); the cookie-header assertions and the concurrent-losers run it asks for are the red tests |
| rv4-FR-F159 (Low) | `DEFERRED_OWNER` | `bd0d020` | `7f0a641` | (b), the loser wiping the winner's cookies, is fixed. (a), a retry whose response was lost succeeding once, is the owner question below; not implemented |
| rv4-FR-F24 (Low) | `DEFERRED_OWNER` | `bd0d020` | `7f0a641` | its "or at least no Set-Cookie with Max-Age -1" alternative is done; "a retry within the grace gets 200" is the same owner question; not implemented |

One red commit serves all four rows: the three red tests are the test sketches
of F100 (two tabs, one jar), F108 (replay by cookie, concurrent losers) and F24
(replay inside the grace, no deletions), played by a cookie jar that applies
`Set-Cookie` in arrival order.

## Owner question (F24, F159): should a retried refresh inside the grace succeed?

A client whose rotation response was lost holds a spent token. Its retry, inside
the 10 s grace, gets 401 and the user signs in again. That is F159 (a) and
F24's scenario, and the cookie fix does not change it: the loser no longer
wipes anything, and it gets no pair either. Letting the retry succeed means
handing a pair to a spent token, which widens the window in which a copied
token is worth something, so it is a security and UX trade-off for the owner.

1. **Keep the 401 (shipped).** Nothing new to keep or mint; a spent token is
   worth nothing in the grace or after it. A lost response ends in a sign-in.
2. **Answer the retry with the winner's pair.** The server stores only the
   SHA-256 of a refresh token, so it would have to hold the successor's raw
   token for about 10 s, encrypted at rest (a new secret to manage) or in
   per-replica memory (which fails across replicas). Whoever holds a copy of
   the spent token, a thief as much as a retrying owner, is handed a live pair
   for 10 s after every rotation, and the family kill cannot tell them apart
   in that window.
3. **Mint a sibling successor for the retry.** No raw token is kept; a retry
   inside the grace gets a fresh session of the same family. The thief gets one
   too (same widening), and one rotation now leaves two live heads in a family.
4. **A client answer**: single-flight across tabs and processes, a lock file
   beside the CLI credentials. It settles the two-tab race and cannot recover a
   response that never arrived. It belongs to the apiclient and Flutter
   clusters and is not a server decision.

Leaning, not implemented: keep 1. `renewalGrace` (500 ms) already covers the
race between a client's own processes, and what is left is a response dropped
on the network, which a sign-in repairs. If the owner wants it closed, 3 is the
smaller change (no secret at rest) and wants a threat-model line in
[backend.md](../../architecture/backend.md) about the widened window.

## Design

**The decision lives in one new store query.** F100's suggested fix is a
tri-state from `RevokeFamilyOnReuse`. The shipped shape is an additive method,
`Store.SpentWithinGrace`: was this token revoked, and no more than `grace` ago?
It is the exact complement of the family kill's edge (`revoked_at >= now() -
grace` against `revoked_at < now() - grace`) on the same database clock, so
`RevokeFamilyOnReuse` keeps its signature and its tests, which neighbouring
clusters call. The two queries need no transaction because `revoked_at` is
write-once (every `UPDATE ... SET revoked_at` carries `AND revoked_at IS
NULL`): a token past the grace stays past it, so asking first and killing
second cannot contradict itself. The price is one point read on the unique
`token_hash` index per refused refresh, a path that is rare by construction.

**The conflict branch touches nothing.** `RotateSession` returns `ErrConflict`
when the session was revoked between the handler's lookup and its own `UPDATE`,
a matter of milliseconds, so always inside the grace. The old code cleared the
cookies there and did not end the lineage; it now does neither, which is what
the lookup-miss branch does inside the grace. If the competing revocation was a
logout, that response already clears the cookies; if it was a family kill or a
password reset, the tokens in the jar are dead and the next refresh with them
meets the first branch (bare for 10 s, then cleared).

**Status and body do not change.** Every cause is still 401 "refresh token
expired or already used", which the Go client's `renewalGrace` path and the
Flutter `refreshEndsSession` read. Outside the grace nothing changes at all:
the cookies are cleared and the family ends, as in wave 3.

**Failure mode.** If the grace read errors, the handler logs it and answers as
for a token past the grace: it attempts the family kill and clears the cookies,
which is the behaviour before this change. A failing database costs the
cookies, not the kill.

## What changed

- **A refused refresh costs what its token deserves** (red `bd0d020`, fix
  `7f0a641`). `server/internal/api/auth.go` gains `refuseNotLiveRefresh`, called
  from the lookup-miss branch of `handleRefresh`: a token spent inside
  `refreshReuseGrace` costs nothing (no `Set-Cookie`, no lineage kill); any
  other token, unknown, expired unrevoked or long spent, gets
  `endLineageOnReuse` and `clearAuthCookies` exactly as before. The
  `ErrConflict` branch no longer calls `clearAuthCookies`. The comment on
  `refreshReuseGrace` now says the loser gets a bare 401.
- **`Store.SpentWithinGrace`** in `server/internal/store/sessions.go`:
  `(ctx, tokenHash, grace) (bool, error)`; unknown, live and expired-unrevoked
  tokens are not spent.
- **Tests.** `server/internal/api/rv4_review_auth_refresh_test.go` (new) plays
  a browser: a jar that applies `Set-Cookie` in arrival order, winner first and
  every loser after it, the worst order the network can deliver.
  `TestRvAuthRefreshReplayInsideTheGraceLeavesTheCookiesAlone` is the
  sequential form (a tab holding a token that was just spent).
  `TestRvAuthRefreshLosingTheRotationRaceLeavesTheCookiesAlone` forces the
  `ErrConflict` branch on the real database with no hook in the handler: the
  test holds `SELECT ... FOR UPDATE` on the session row, the refresh finds the
  session live and parks in `RotateSession`'s `UPDATE` on that lock (seen in
  `pg_stat_activity`), the test revokes the row, stores the successor and
  commits, and the parked `UPDATE` wakes to zero rows.
  `TestRvAuthRefreshConcurrentTabsOnOneJarKeepTheWinnersPair` fires eight
  refreshes with one jar at once, requires exactly one 200, a bare 401 for the
  seven others whichever branch they took, and that the winner's pair still
  refreshes afterwards. Two guards, green on the base and still green:
  `TestRvAuthRefreshReplayAfterTheGraceStillEndsTheFamilyAndClearsTheCookies`
  (a replay after the grace clears the cookies it was sent and ends the family)
  and `TestRvAuthRefreshTokenThatIsNotASpentOneIsStillCleared` (an unknown token
  and an expired never-revoked one are cleared). In
  `server/internal/store/sessions_family_test.go`, two tests for the new method
  ride in the fix commit because they cannot compile without it: the edge
  (revoked 0 s and 5 s ago inside a 10 s grace reads true and the family kill
  ends nothing; 15 s ago reads false and the kill ends one) and which tokens
  count (live, expired unrevoked and unknown read false; logged-out reads true).

## Evidence

Red, at `bd0d020` on the base `auth.go` (private PostgreSQL): three `FAIL`s. The replay: "the replay inside the grace: the 401 sets or deletes
insideout_access (Max-Age -1) ..." and the same for `insideout_refresh`, then
"after the loser's response the browser holds access cookie false, refresh
cookie false, want the winner's pair". The forced conflict: the same two
lines for "the loser of the rotation race". The eight tabs: "loser tab 0" to
"loser tab 6", seven losers, each deleting both cookies, and the browser ends
with neither. The two guards pass on the base.

Green, at `7f0a641`: the five `TestRvAuthRefresh*` api tests and the two store
tests pass with `-race -count=1`; the neighbouring grace tests
(`TestRefreshReplayInsideTheGraceLeavesTheWinnersPairAlive`,
`TestRefreshReuseAfterTheGraceEndsTheWholeFamily`,
`TestConcurrentRefreshesOfOneTokenLeaveOneWinnerAlive`) are untouched and pass.

Mutations, each applied to the fix and reverted (the failing set is the new
tests only; names are shortened, all are `TestRvAuthRefresh...`):

| Mutation | Tests that go red |
| --- | --- |
| lookup-miss branch back to `endLineageOnReuse` + `clearAuthCookies` | `ReplayInsideTheGrace`, `ConcurrentTabsOnOneJar` |
| `ErrConflict` branch clears the cookies again | `LosingTheRotationRace` |
| `>=` flipped to `<` in `SpentWithinGrace` | both store tests, and in the api package `ReplayInsideTheGrace`, `ConcurrentTabsOnOneJar`, `ReplayAfterTheGrace` |
| `if inGrace` always true (never end the lineage or clear) | `ReplayAfterTheGrace`, `TokenThatIsNotASpentOne` |

No live server was started: the routes are driven in-process over the real
database, and the jar is a model of the browser's apply-in-order rule, not a
browser.

## Ceilings

- **A lost response still ends in a sign-in** (F159 (a), F24): the owner
  question above.
- **A Flutter tab that loses still ends its own session** on the 401
  (`refreshEndsSession`, `client/lib/api/sse.dart`), a client rule another
  cluster owns. What changes is that the shared cookie jar is no longer emptied
  by it.
- **A bare 401 is distinguishable from a cleared one.** A presenter can tell a
  token spent in the last 10 s (no `Set-Cookie`) from an unknown or older one
  (cookies cleared). The status and body are identical, and what it reveals,
  that the token was rotated recently, the holder of the token already knows.
- **Other revocations ride the same 10 s.** A session revoked by logout, a
  password reset or a family kill also reads as "spent inside the grace": for
  10 s afterwards a refresh with its token gets the bare 401, and after that the
  usual clear (the kill is then a no-op). A browser can hold a dead pair up to
  10 s longer on that path; logout's own response already clears it.
- **The browser's ordering is modelled, not observed.** A real two-tab run in a
  browser was not made.

## INTEGRATOR TODO

- `server/internal/api/coverage_session_test.go`: gate-needed.
  `TestRefreshTakesItsTokenFromTheCookieOrTheBody` asserts at line 147 that a
  replay of a just-spent token clears both cookies, under a comment that says
  "its refusal does not cost the winner's pair"; the assertion is the defect
  this cluster fixes. Without the patch the `./internal/api/...` gate fails on
  that one test. With it the whole api package passes (proved with `go test
  -overlay`, the file itself untouched):

  ```diff
  --- a/server/internal/api/coverage_session_test.go
  +++ b/server/internal/api/coverage_session_test.go
  @@ -144,7 +144,13 @@
   	if rec.Code != http.StatusUnauthorized {
   		t.Fatalf("a spent token refreshed again: %d %s", rec.Code, rec.Body.String())
   	}
  -	expectCleared(t, "refresh with a spent token", rec)
  +	// Inside the reuse grace the 401 sets and deletes nothing: the cookies in the
  +	// browser are the winner's now (rv4-FR-F100).
  +	for _, name := range []string{accessCookieName, refreshCookieName} {
  +		if c := setCookie(rec, name); c != nil {
  +			t.Fatalf("a spent token inside the grace: Set-Cookie %s = %+v, want the winner's cookies left alone", name, c)
  +		}
  +	}
   
   	// The cookie wins when the body carries something else.
   	rec = refresh(`{"refreshToken":"garbage"}`, &http.Cookie{Name: refreshCookieName, Value: next.RefreshToken})
  ```

- `docs/architecture/backend.md`: line 84, after "inside the grace is only a
  lost race (rv4-SEC-04)", add that the replay is refused with a 401 that sets
  and deletes no cookie, so the tab that lost leaves the winner's cookies alone
  (rv4-FR-F100).
- `docs/changelogs/2026-10-18-rv4-remediation/index.md`: add the
  `w4-rv-auth-refresh` row to the cluster-fragment table.
- `docs/issues/2026-10-18-review-4-open-findings.md`: record the F24 and F159
  owner question (a retried refresh inside the grace answering 200, options 1
  to 4 above) as open residue.

## Gates

The cluster's seven gates were run from this branch. All but one are clean; the
exception is `go test ./internal/api/...`, red on the single test named above
until the integrator applies the patch, green with it. The store and apiclient
packages are green as they stand.
