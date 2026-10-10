# Contributing to InsideOut

InsideOut is MPL-2.0 ([LICENSE](LICENSE)) and runs a hosted multi-user
instance, so outside contributions are welcome. This page is only the door:
the working contract is [AGENTS.md](AGENTS.md) — read it before your first
change. It is written for the coding agents this repo is developed with, and
it is also where the repo keeps its rules in full; nothing here repeats or
overrides it.

## Before you open a PR

- Read [AGENTS.md](AGENTS.md), then start from the
  [doc map](docs/index.md) for the area you are touching.
- `docs/` prose is English-only.
- DB-dependent tests run against real PostgreSQL and must run, not skip —
  no mocks. Local setup:
  [docs/usage/local-development.md](docs/usage/local-development.md).
- Client changes follow the visual rules in [DESIGN.md](DESIGN.md).

## Recording the change

Meaningful changes get a dated entry under
[docs/changelogs/](docs/changelogs/README.md) — one dated file for an
ordinary change, a dated folder for a large one, and the nearest index
updated. The local hook only warns and CI does not enforce this, so the PR
author adds the entry. `docs/plans/` and `docs/HANDOFF.md` are the
maintainer's running records; leave them to the maintainer.

## CI gates a PR must pass

[.github/workflows/ci.yml](.github/workflows/ci.yml) runs on every PR: the
server suite against real PostgreSQL (gofmt, build, vet, `go test ./...
-race` — a skipped test is a red build), the server image build, the nginx
config checks, the Flutter gates (`flutter analyze` under an info baseline,
`dart format`, `flutter test` including the browser suite, `flutter build
web`), a release-APK build, and the env-catalog suites with the blocking
redaction guard.

## Security

Vulnerabilities in the hosted instance or the code go to the maintainers
privately — see [SECURITY.md](SECURITY.md). Please do not open a public
issue for them.
