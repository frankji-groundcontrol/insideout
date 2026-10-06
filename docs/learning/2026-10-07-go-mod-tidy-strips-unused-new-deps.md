# 2026-10-07 — go mod tidy silently strips newly added but not-yet-imported deps

## What was learned

`go get module@latest` followed by `go mod tidy` **removes** modules
that no package imports yet — the exact opposite of what a
"deps-first commit" for upcoming parallel work needs. The first run
of the setup-TUI deps step added bubbletea/huh/x-term and then tidied
them straight back out; only a follow-up `grep … go.mod` verification
caught it, because `go get`'s output ("added …") reads like success.

## Evidence

[Setup TUI plan](../plans/2026-10-07-setup-tui.md), deps step
(2026-10-07): the first `go get github.com/charmbracelet/bubbletea@latest
github.com/charmbracelet/huh@latest golang.org/x/term@latest && go mod
tidy` printed additions, then tidy dropped every not-yet-imported one;
re-running `go get` **without** tidy left them in `go.mod` as
`// indirect` requires, build/vet/tests green.

## Scope

Any change that pre-lands dependencies before the code that imports
them lands — the parallel-workflow pattern where a lone deps commit
precedes fan-out workers. Also: deps stay marked `// indirect` until
imported; a `go mod tidy` at join time (when imports exist) is what
flips them to direct requires.

## When to apply

In a deps-first step: run `go get` and **skip `go mod tidy`**, then
verify with `grep <module> go.mod` and a build — never trust the
`go get` output line alone. Schedule the real `go mod tidy` for after
the importing code lands.
