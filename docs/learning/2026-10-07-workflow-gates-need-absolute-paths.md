# 2026-10-07 — A workflow gate must use absolute paths: world.run's cwd is not guaranteed to be the repo root

## What was learned

`world.run` executes with the **workspace** as cwd — and the
workspace of a GUI-amended run can differ from the repository root
(the settings-change run `dwfrun-3453f271` executed its gate with
cwd `<repo>/scripts`). Every relative path in a gate command then
fails before any code runs, and the failure masquerades as a code
defect: `go: chdir server: no such file or directory`,
`python3: can't open '.../scripts/scripts/test_env_catalog.py'`
(the doubled segment is the tell — a cwd off by one directory).

## Evidence

The setup-TUI workflow's gate-fixer escalated with exactly this
(`dwfq-3453f271-1`): two identical gate rounds failing on path
resolution while its own re-runs from the correct directories passed
everything (25 Go packages ok, 47+37 env battery tests, gofmt clean,
zero files edited). The doubled `scripts/scripts/` path pinned the
cwd to `<repo>/scripts`.

## Scope

Every dynamic-workflow script that gates on `world.run` —
especially ones resumed/superseded from the GUI, where the workspace
binding can shift.

## When to apply

Write gate commands cwd-independent from the start: literal command
names with **absolute paths in the args array** (`<repo>` is the
absolute checkout root: `go -C <repo>/server`, `python3
<repo>/scripts/test_env_catalog.py`,
`gofmt -l` with absolute paths, `stat` with an absolute path). When
a gate fails with "no such file or directory" on files that visibly
exist, check the cwd theory before hunting code defects — and give
repair workers a scope narrow enough that "this failure cannot
belong to my files" is a conclusion they can reach and escalate, as
happened here.
