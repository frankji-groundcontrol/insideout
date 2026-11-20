# 2026-11-21 — govulncheck red on main: Go toolchain behind two stdlib fixes

## Symptom

The `security` workflow on the first push of wave 4b
(`4d186da9`, run 44786716778, 2026-11-20 UTC) failed in
`govulncheck (server)`; `gitleaks` passed. Two reachable standard-library
vulnerabilities:

- GO-2026-6859 — `net/http`, found in go1.26.9, fixed in go1.26.11
  (`insideout.runServe` calls `http.Server.ListenAndServe`).
- GO-2026-6809 — `crypto/x509`, found in go1.26.9, fixed in go1.26.10.

## Root cause

`server/go.mod` pins `toolchain go1.26.9`; CI builds with that toolchain.
The vulnerabilities are in the Go standard library, not in a module
dependency — no `go get` bump fixes them.

## Bounded fix

Raise the `toolchain` directive in `server/go.mod` to `go1.26.11` (the
`go 1.26.0` floor and the five Go-floor texts are unaffected), check the
Dockerfile and CI for a separate Go version pin and move it with the
toolchain, then confirm `govulncheck` is green in CI. Local hosts need the
matching toolchain (Go downloads it automatically via `GOTOOLCHAIN`).

Found during the [2026-11-21 sync and push](../changelogs/2026-11-21-wave-4b-sync-and-push.md);
the `ci` workflow on the same commit was still in progress at recording
time.