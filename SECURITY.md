# Security Policy

InsideOut is an MPL-2.0 application with a hosted multi-user deployment: the
API the CLI calls by default plus the Flutter web client in front of it — the
deployment whose base URL ships as `DefaultBase` in
[server/internal/apiclient/default_base.go](server/internal/apiclient/default_base.go).
This page is the disclosure door for that hosted instance and for the code in
this repository, in good faith and without a fixed timeline.

## Reporting a vulnerability

Report privately — please do not open a public issue:

- **GitHub private security advisory (preferred):**
  https://github.com/frankji-groundcontrol/insideout/security/advisories/new
- Failing that, contact the repository owner through GitHub
  (`frankji-groundcontrol`) — the same contact the README's install path
  already exposes.

Include what you found, steps or a proof of concept, the affected URLs or
files, and the impact you see. Reports are acknowledged and kept private
while triaged; disclosure is coordinated with the reporter after a fix
ships.

## Scope

In scope:

- The hosted multi-user instance described above — its API, web client, and
  the account / auth / workspace flows it runs — including what an
  unauthenticated visitor or an ordinary signed-in user can reach.
- The code in this repository: the Go server (`server/`), the Flutter client
  (`client/`), the CLI, and the nginx configuration it ships.

Out of scope:

- Self-hosted deployments you run yourself — report those to your operator.
  A defect in the code that only shows up when self-hosting is still in
  scope.
- Third-party services and libraries (report those upstream), and
  automated-scanner reports without a demonstrated impact.

## Review records

Whole-tree review reports, each with a security findings file
(`22-security.md`), are kept under [docs/review/](docs/review/README.md).
