# 2026-11-21 — Close the gaps the review 5 remediation left open

Plan: [close the review 5 gaps](../plans/2026-11-21-close-review-5-gaps/2026-11-21-close-review-5-gaps.md).
Four gaps were left open by the
[review 5 remediation](./2026-11-21-rv5-remediation/index.md). They shared no
file, so four clusters ran in parallel from `main` at `cecb0cb`. The
coordinator reviewed each diff and integrated it on `rv5/close-gaps`.

| Gap | What changed | Commits |
| --- | --- | --- |
| Long functions | All eight functions left over 80 lines are split into named steps. Five are in the store, setup and LLM code, three are API handlers. The store helpers run in the caller's transaction, behind the same advisory locks. The function guard's list is empty, so any function over the threshold now fails it ([record](../issues/2026-11-21-long-functions.md), resolved) | `c0e891a1`, `0f9aadbd` |
| Store errors mapped by hand | Every remaining store-error ladder in `server/internal/api` answers through `writeStoreError` (115 calls). `ErrRepoTaken` keeps its `github_repo_already_linked` code. 38 route cases pin status, code and message. One behaviour change: removing or demoting the last admin answers 409 `last_admin` instead of `conflict`, matching the other routes, and no client read the old code. The residue is 35 deliberate checks, each commented ([record](../issues/2026-11-21-store-error-ladders.md), resolved) | `fe53f836`, `55598686`, `bad81fbf`, `4b5f1630`, `39d982c2` |
| Per-host key lists | One shared manifest set under `hosts/_manifests/keys/`; each host keeps its `host.yml`, README and ignored `.secrets/`. `keys.key_map` reads the shared set. The manifests now map exactly the variables `.env.example` declares: `INSIDEOUT_AI_TURN_DEADLINE` and `INSIDEOUT_TRUSTED_PROXY_HOPS` were added, and `INSIDEOUT_GH_CLIENT_SECRET`, which no code reads, was unmapped ([record](../issues/2026-11-21-key-manifests-one-set.md), resolved) | `4fbfae0a`, `35b979cc`, `4c1b18b0`, `8dab7301`, `3920f274`, `60885b2f` |
| Parity test reads top-level fields only | The parity walk now reads nested request fields (proposal items, tree nodes) down to the field that fills each one. It counts a client call by the receiver type that declares the method, so a same-named method elsewhere no longer counts. It found no new gap ([practice](../practices/2026-11-21-surface-parity-guards.md)) | `13e86b02`, `b7a5c501`, `f95ebe3f`, `6c270087`, `e5e213fa` |

## Operator note

The two new keys need a `.key` placeholder in each machine's store. Run
`INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py init`. It creates empty
files only (an empty file means the default is in effect), never changes an
existing file and never prints a value. It was run on the coordinator's
machine, where all 57 key tests pass. The old `gh-client-secret.key` file
stays on disk; nothing reads it.

## Verification

All gates ran locally on the integrated branch. CI is disabled in the
repository.

- `go build`, `go vet` and `gofmt`: clean.
- `go test ./... -count=1`, without a database and on a freshly migrated
  private PostgreSQL 18 database: every package ok, no skipped tests.
- `flutter analyze`: 8 infos (baseline 8). `dart format`: no change.
  `flutter test`: 667 passed.
- The key suite (`scripts/tests/keys`), every `scripts/tests/check_*.py` and
  `scripts/check-redaction.sh`: exit 0.

## Left open

These limits of the parity walk are recorded in the practice:

- a type nested inside itself is walked once;
- the JSON inside a forwarded raw argument is not checked;
- struct field assignments still match by file and field name;
- calls made through a method value or an interface are not counted.
