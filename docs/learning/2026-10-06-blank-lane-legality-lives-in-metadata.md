# A blank placeholder is only legal if metadata says so — declare the lane first

## What was learned

A never-absent scheme ("every mapped kind has a file; blank = default
in effect") converts an *implicit* state — the file simply not
existing on a host — into an *explicit* one: an empty file. That
explicitness immediately asks a question the old scheme never had to
answer: **is empty actually legal here?** The answer must live in
metadata (`fallback` / `optional` / `operator_fills` fields in the
manifest), and it must be declared **per host before placeholders are
materialized** — otherwise the enforcement suite fails on hosts that
legitimately run the blank lane. Legality of a value's absence is a
per-host fact, not a global one: the same kind is filled on one host
(online coach) and blank on another (offline coach), and both are
correct.

## Evidence

Extending the key store to every env var (2026-10-06,
[plan](../plans/2026-10-06-config-companion-keys.md)) added
`keys.py init`, which creates empty `.key` placeholders for all mapped
kinds. Buddy passed its 46-test suite immediately. The first MBP run
failed 5 tests: `init` had created empty files for the phase-1
companion kinds (`llm-base-url`, `llm-model`, `llm-schema`,
`gh-private-key-file`) whose manifest entries carried no
`optional`/`fallback` field — MBP legitimately runs them blank
(offline coach; env-embedded PEM lane), but nothing *declared* that,
so the store invariant read the empties as unmapped-but-required and
the shape tests read them as broken values. Fix: mark the four entries
`optional: true` on MBP's own manifests and make the four shape tests
skip a blank file as "unset lane on this host" (commit `40635cf`);
suite then 46 OK on both hosts. The `fallback`-presence variant of the
same rule was caught test-first on Buddy the same day
(`fallback: ''` for cors-origins — key *present* with an empty value
is a legal blank; checking truthiness instead of presence inverted the
meaning).

## Scope

Any scheme that materializes placeholders for optional configuration
per host/machine/tenant: key stores, dotfile managers, deployment
scaffolds. Whenever "absent" becomes "present-but-empty", pair the
materializer with a metadata field that licenses emptiness, and derive
the emptiness rule from the field's **presence**, not its truthiness.

## When to apply it again

Before running any placeholder-creating step (`keys.py init` or
equivalent) on a new host, check every mapped kind that host will hold
blank carries `fallback`/`optional`/`operator_fills` in *that host's*
manifest — the failure mode is the invariant suite, not the
materializer, reporting it.
