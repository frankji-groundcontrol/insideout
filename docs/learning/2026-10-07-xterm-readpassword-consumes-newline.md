# 2026-10-07 — x/term ReadPassword already consumes the trailing newline

## What was learned

`golang.org/x/term.ReadPassword` (v0.46.0, `terminal.go:1055`,
`1062-1065`) reads through and **consumes** the newline that ends the
secret — it does not leave it on the fd. A hidden-input helper that
then "drains the trailing newline" from a shared buffered reader
hangs: the second Enter waits for input that will never arrive
because the first was already fully consumed.

## Evidence

The setup-TUI workflow's independent review of the hidden-prompts
slot (run `dwfrun-3453f271`, reviewer-C) caught exactly this at
`server/cmd/insideout/cli_auth.go:190` in the first cut: the TTY path
hung for a second Enter per secret. The fix removes the drain after
`ReadPassword`. This is the TTY twin of the shared-stdin-buffer trap
recorded 2026-08-24
([learning](2026-08-24-cli-prompts-share-one-stdin-reader.md)): the
buffered reader still matters for the non-TTY path, but after
`ReadPassword` there is nothing left to drain.

## Scope

Any Go code that pairs `x/term.ReadPassword` with a shared buffered
stdin reader — login/password prompts, code entry, token pastes.

## When to apply

When writing a dual-mode (TTY/pipe) secret reader: branch on
`term.IsTerminal`, call `ReadPassword` on the TTY path and return its
string as-is, and keep the buffered reader only on the piped path.
Never add a post-`ReadPassword` newline drain, and never mix the two
readers on one path.
