# 2026-11-21 — key manifests: one set under hosts/_manifests/keys/

*Resolved 2026-11-21. Found by review 5 as `STRC-04`
([findings register](../review/2026-11-21-focused-review-5/01-findings-register.md)).
Wave 4 landed the guard only; the move is this record.*

## Symptom

Every host folder under `hosts/` carries its own copy of the key manifests
(`hosts/<host>/keys/<module>/*.yaml`). The two host trees hold 12 manifests
each that differ only in the `host:` line and three `optional: true` lines.
The copies are a second catalogue of the variables in `.env.example`, and
they have drifted from it:

- `.env.example` declares `INSIDEOUT_AI_TURN_DEADLINE` and
  `INSIDEOUT_TRUSTED_PROXY_HOPS`; no manifest maps either.
- `github/app.yaml` maps `INSIDEOUT_GH_CLIENT_SECRET` ("no server reader
  yet"), which `.env.example` does not declare.

## Guard already in place

`test_manifest_variables_match_the_skeleton` in
[`scripts/tests/keys/test_store.py`](../../scripts/tests/keys/test_store.py)
asserts, for every host, that the mapped variables equal the names
`.env.example` declares, with the three differences above listed as
`SKELETON_ONLY` and `MANIFEST_ONLY`. It needs no secret values, so it runs
in CI's env-suites job. Any new drift fails it.

## Why the move was not done in wave 4

The brief capped the move at about 150 changed lines. Moving the manifests
changes `scripts/keys.py` (`key_map` reads the manifests from one place and
resolves `.secrets/` under the host), the fixtures in
`scripts/tests/keys/test_tool.py` and `_helpers.py`, both per-host
`keys/README.md` status tables, and about 38 living docs that name
`hosts/<host>/keys/<module>/` paths (mostly `docs/usage/keys/`).

## Bounded fix

1. `git mv` one host's `keys/<module>/*.yaml` to
   `hosts/_manifests/keys/<module>/`, drop the `host:` line, keep
   `optional: true` where either copy had it; delete the other host's copies.
   `host.yml`, `keys/README.md` and the ignored `.secrets/` stay per host.
2. `keys.key_map(root, host)`: glob `hosts/_manifests/keys/*/*.yaml`, resolve
   each value file to `hosts/<host>/keys/<module>/.secrets/<name>.key`.
   `discover_hosts` already skips `_manifests` (it has no `host.yml`).
3. Point `test_store.manifest_entries`, `test_tool.make_tree` and the docs at
   the new manifest path; the `.secrets/` paths do not change.
4. Settle the drift: add manifests (plus the per-key doc row and test the
   store's README asks for) for `INSIDEOUT_AI_TURN_DEADLINE` and
   `INSIDEOUT_TRUSTED_PROXY_HOPS`, decide whether `INSIDEOUT_GH_CLIENT_SECRET`
   belongs in `.env.example`, and empty both exception sets.

Verify with `python3 -m unittest discover -s tests/keys -t . -p "test_*.py"`
from `scripts/`, and `python3 -I scripts/keys.py audit` on a host that has a
store.

## Resolution (2026-11-21)

- One manifest set now lives under
  [`hosts/_manifests/keys/`](../../hosts/_manifests/keys/): the primary host's
  copies were moved (they carried the three `optional: true` lines), the
  `host:` line and host-specific notes dropped, and the other host's copies
  deleted. `host.yml`, `keys/README.md` and the ignored `.secrets/` stay per
  host; the `.secrets/` paths did not change.
- `keys.key_map(root, host)` reads the shared manifests (`keys.manifest_paths`)
  and resolves each value to `hosts/<host>/keys/<module>/.secrets/<name>.key`.
  `test_manifests_live_in_one_shared_set` fails if a per-host copy returns.
- Drift settled: `INSIDEOUT_AI_TURN_DEADLINE` is mapped as
  `llm/coach.yaml` `ai-turn-deadline` and `INSIDEOUT_TRUSTED_PROXY_HOPS` as
  `app/runtime.yaml` `trusted-proxy-hops`, each with a per-key doc and test.
  `INSIDEOUT_GH_CLIENT_SECRET` is **unmapped, not added to `.env.example`**:
  no server code reads it, so it is not a variable of this app; its per-key
  doc and test were removed. Both exception sets in `test_store.py` are
  empty.
- Operator follow-up on each machine with a store: run
  `INSIDEOUT_KEYS_HOST=<host> python3 scripts/keys.py init` to create the two
  new placeholders (the never-absent test fails on a present host until
  then). The old `gh-client-secret.key` stays on disk unread; delete it or
  keep the value in a password manager if the OAuth flow is ever built.
