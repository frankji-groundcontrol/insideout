# A boot-time warning is not a pass — grep for silent degradations

## What was learned

A service that boots "successfully" can boot **degraded**: optional
features fail at init, log a `Warn`, and continue. Treating "the
process started" as verification silently accepts the degraded state.
Verification of any credential must assert the *absence of the
degradation warning*, not just process liveness — and canonical stored
forms must match how each consumer reads them (raw file vs env-escaped).

## Evidence

The GitHub App private key on the new host store (2026-10-05) was
copied as the `\n`-escaped single-line **env form**.
`github.LoadPrivateKey` reads the `_FILE` variant **raw** (no
unescaping — only the env-var path unescapes), so `pem.Decode` failed,
`server.go` logged
`github app private key unusable — guide loads will be unauthenticated`,
and the server ran fine — with the evidence loop silently disabled.
Two boot probes counted as "config accepted" while that warning sat in
the same logs. The per-key unit suite (`scripts/tests/keys/`) caught it
via the PEM-shape assertion; normalizing the file (1702→1675 B) and
re-booting made the warning disappear — that disappearance, not the
boot, is the pass condition. The same defect then turned out to exist
on the primary host's store too, plus two more MBP-local findings
(0755 `.secrets` dirs; wrong-case host folder on a case-insensitive
filesystem) — all found by running the suite on the second host, not
by writing more docs
([changelog](../changelogs/2026-10-05-buddy-host-keys.md)).

## Scope

Any config with warn-and-continue semantics (feature keys, optional
integrations, _FILE/_env dual consumers). Also: run new test suites on
every host they describe — the first host passing proves less than it
appears.

## When to apply it again

Verification steps must name the exact log line whose **absence** is
the pass; per-key docs under `docs/usage/keys/` encode this. When a
value has two storage forms (escaped/env vs canonical/file), each
store holds the form its consumer reads raw — and a copy between
consumers is a transformation, not a byte copy.
