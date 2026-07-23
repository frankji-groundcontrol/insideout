# Unmaintained LLM abstraction libraries: own a small direct client instead

**Date**: 2026-07-21

## What was learned

An abstraction library's job is to hide provider wire-format differences.
When the library itself gets the wire format **wrong**, the abstraction
actively works against you, and no amount of application-level
engineering-around can fix it without reading the library's internals.

Concretely: `langchaingo v0.1.14`'s Anthropic provider maps every response
content block to a separate `resp.Choices[i]` — one choice per *content
block*, not one per completion candidate as most providers use the field. An
extended-thinking model returns a `thinking` block before the real `text`
block, so code reading `Choices[0]` got an empty result (`no response`) while
the actual answer sat unread at `Choices[1]`. This was the *second* real
wire-format bug in the same provider (the first: dropped parallel tool
calls), in a dependency whose last commit predated the session by months.

The resolution that held: for a **single-provider integration on an
actively-evolving API surface** (extended thinking is recent), owning a
small, well-tested direct client is lower risk than an unmaintained
dependency. The Anthropic Messages API is a small, stable, documented
surface; the replacement (`server/internal/agent/anthropic.go`) is stdlib
`net/http` + a hand-rolled SSE parser, with fixture tests built from the
exact live payloads that broke the old code. Keeping the provider behind a
one-method interface (`ChatStreamer`) made the swap a one-file change.

## Evidence

[BUG-009](../issues/2026-07-21-bug-009-langchaingo-removed.md) — root-caused by reading
the vendored library source, confirmed by replaying the identical request
with `curl`.

## Scope

Any dependency that abstracts a third-party wire protocol — LLM SDKs
especially, but the lesson generalizes to any protocol-translation library
whose upstream protocol is evolving faster than the library.

## When to apply again

- Choosing how to integrate a single external API: check the library's
  maintenance pulse (last commit, open "is this dead?" issues) against the
  API's rate of change before adopting it. One provider + small stable
  surface + stale library = write the client.
- Debugging "empty response" style failures from an abstraction layer:
  replay the raw request with `curl` first; if the wire response is fine, the
  bug is in the translation layer, and the fix may not be reachable from
  application code.
- Designing any provider integration: put it behind a minimal interface so
  the implementation is swappable in one file — that is what made this
  removal cheap.
