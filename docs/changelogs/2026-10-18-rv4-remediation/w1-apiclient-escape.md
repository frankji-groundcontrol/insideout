# Wave 1, apiclient escaping — review-4 remediation

Branch `rv4/w1-apiclient-escape`, `1dba3fe..a1bd90e` (two commits: the red
test and the fix), then this fragment. The branch already existed at the base
SHA and was reused. Finding `rv4-QUAL-03`, the apiclient half of row 1.13 of
the [remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the `cooldownClientIP` half (rv4-QUAL-02) belongs to another cluster. Findings
are cited as `rv4-<ID>` because review IDs restart per review. No value, name
or address is spelled in this fragment.

## What changed

- **One helper file builds every apiclient request target** (rv4-QUAL-03, red
  `fe12ffe`, fix `a1bd90e`). `server/internal/apiclient` glued ids,
  audiences, formats and focus values into paths and queries by hand, and the
  MCP tools feed it model-chosen strings. New `escape.go` (49 lines):
  `apiPath` makes each argument exactly one path segment, `withQuery` appends
  query-escaped pairs in call order. 37 client methods in `apiclient.go`,
  `auth.go`, `crud.go`, `prd.go` and `roadmap.go` use them (39 source lines;
  the review counted 36 sites, and the sweep below, not a count, is what
  proves none is missed).
- **What the old code allowed**, shown against a real `http.ServeMux` (the
  server's router type): `Prd("../me")` was answered by `GET /me` under the
  caller's bearer (the mux cleans the dot segment and redirects, and the
  client follows a same-host redirect with its token); `PrdView` with the
  audience `decision&admin=1#frag` sent `audience=decision` plus a smuggled
  `admin=1`; a value with `%` or CRLF made the client fail before it sent
  anything.
- **Valid input is byte-identical.** An ordinary id holds none of the escaped
  characters, and query keys keep their call order, so every URL the
  existing tests pin (`format=markdown&audience=...`, the `project_id`,
  `mode`, `focus` order) is unchanged. Those tests were not edited and pass.

## The escape rule

- A path argument goes through `url.PathEscape` plus the two dot segments
  (`.` becomes `%2E`, `..` becomes `%2E%2E`): `PathEscape` leaves them bare,
  and a server or proxy resolves them.
- A query value goes through `url.QueryEscape`, not `PathEscape`, which leaves
  `&`, `=` and `+` alone. `url.Values.Encode` was not used for new sites
  because it sorts keys and would reorder URLs that valid callers already
  send; `PrdExport` keeps its existing `url.Values` (sorted `audience`,
  `format`), so its bytes do not change either.
- It is a function, not `"/prds/" + url.PathEscape(id)` inline: the plan's
  grep gate (a quoted `/path` followed by `+` and a name) still matches the
  inline form, and one function is the one place the dot-segment rule lives.

## Evidence

All in `server/internal/apiclient/escape_test.go` (310 lines). It uses the
public API only, so it compiles and fails on the unfixed code.

- **Red** `fe12ffe`, replayed alone from a pristine `git archive` of that
  commit: exit 1. `TestHostileValuesStayOneSegmentOrOneQueryValue` fails on
  39 of its 57 rows and passes the other 18 (the methods with a fixed path);
  `TestHostileValuesReachTheirOwnRoute` fails; the completeness test passes.
- **Green** `a1bd90e` under `-race`: 60 PASS lines (3 tests plus 57 rows), no
  FAIL. The outputs are kept as `w1-apiclient-escape-red-rv4-QUAL-03.txt` and
  `w1-apiclient-escape-green-rv4-QUAL-03.txt` under
  `/private/tmp/insideout-rv4/reports/`.
- **Correction.** The red commit message says "the 17 fixed-path rows"; the
  count is 18 (57 rows, 39 red). History is not rewritten, so this line is the
  correction.
- **The table**: 12 hostile inputs (`../x`, `a/b`, `a?b=c#d`, `a b`, `%2e%2e`,
  `..`, `.`, `decision&admin=1#frag`, `a+b`, `100%`, CRLF, non-ASCII) go into
  every id, audience, format, mode and focus argument. The oracle reads the
  request as it arrived on the wire (`RequestURI`), not the decoded `URL.Path`,
  which would hide exactly this bug. 57 rows cover 55 methods (`AgentContext`
  and `PrdExport` have two each); `SetToken` and `Token` build no URL and are
  exempt.
- **Completeness is a test.** `TestSweepCoversEveryClientMethod` lists the
  exported `*Client` methods by reflection and fails for any without a row, so
  a method added later cannot skip the sweep.

Mutation checks (each applied to the fix commit, then `git checkout --`; the
tree was clean afterwards). A verifier can repeat them:

| Mutation | What goes red | FAIL lines |
| --- | --- | --- |
| `Prd` back to `"/prds/" + id` | sweep, router test | 3 |
| `escSeg` without the dot-segment rule | sweep | 38 |
| `withQuery` escapes values with `PathEscape` | sweep, router test | 6 |
| `PrdExportAudience` query order flipped | the existing `TestPrdExportAudienceKeepsURL`, sweep | 3 |
| `apiPath` writes segments raw | sweep, router test | 39 |
| `AgentContext` focus smuggles `&x=1` | sweep | 2 |
| a new exported `Client` method with no row | completeness test | 1 |
| the `RoadmapDelete` row deleted | completeness test | 1 |

## Known ceilings

- Client side only. The client does not validate ids (no shape check): a
  malformed id is escaped into one segment and the server answers 404 or 400
  as for any unknown id. `c.base`, the operator's own API address, is used as
  given; only values a caller supplies are escaped.
- Fixed paths (`/auth/...`, `/admin/invitations`, `/workspaces`) carry no
  caller value and stay literals. Their 18 rows are in the sweep so a later
  edit that adds a value to one is caught.
- `Cursor` passes `nil` as the reply target, so a JSON reply would make it
  error after the request went out. The real route answers 204, so it does
  not bite; the sweep row ignores the reply error and the code is unchanged.
- The `prd.go` locale parameter is wave 3's and was not added. When it lands
  it is one more `withQuery` pair, and the compiler forces the sweep rows for
  `PrdView` and `PrdExport*` to gain the argument.

## Verification

Gates, all on `a1bd90e` with the fragment staged:

- `gofmt -l server/internal/apiclient/` prints nothing; `go build ./...` and
  `go vet` on the apiclient, mcpserver and cmd trees are clean.
- The plan's grep gate, `git grep -nE '"/[a-z0-9/-]*" *\+ *[a-zA-Z]' --
  server/internal/apiclient`, prints nothing.
- `go test ./internal/apiclient/... ./internal/mcpserver/... ./cmd/...
  ./internal/setup/... -count=1 -race` passes (every caller of the client).
- `go test ./... -race -count=1` ran once as the last code gate: 25 packages
  ok, 3 without test files, and one failure outside this cluster,
  `internal/api` `TestProjectQuietFlagged` (2 subtests). It fails identically
  on a pristine archive of the base `1dba3fe`: the test pins `now` to a fixed
  date while the view builder reads the real clock. Plan row 1.3
  (`w1-clock-and-runes`) fixes it, and it was left untouched here.
  `DATABASE_URL` was unset, so the DB-gated tests skip; this cluster touches
  no SQL or store code.
- `sh scripts/check-redaction.sh`, `sh scripts/check-docs-recorded.sh` and
  every `scripts/tests/check_*.py` pass.
