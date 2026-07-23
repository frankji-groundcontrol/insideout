# BUG-005: `go.mod`'s `go` directive inherited the local dev machine's toolchain version, breaking the Docker build

**Found**: 2026-07-20, during the InsideOut rewrite (P7), running `docker compose build server` for the first time.

**Symptom**: `go: go.mod requires go >= 1.26.2 (running go 1.24.13; GOTOOLCHAIN=local)`, failing the `go mod download` build step.

**Root cause**: `go mod init` sets the `go` directive in `go.mod` to the version of the toolchain that ran it. The local development machine had Go 1.26.2 installed, so `go.mod` declared `go 1.26.2` even though nothing in the code actually required that version — it was purely an artifact of whichever machine happened to run `go mod init`. The `Dockerfile`'s `golang:1.24-alpine` base image (a widely-available, stable tag) couldn't satisfy that requirement, since Go's toolchain directive enforcement refuses to build with an older toolchain than declared.

**Fix**: lowered the `go` directive to `1.24.0`, then ran `go mod tidy`, which raised it back to `1.25.0` — the actual minimum required by a transitive dependency. Updated `Dockerfile`'s base image to `golang:1.25-alpine` to match. Verified `go build`/`go vet`/`go test` all still pass locally (a newer local toolchain happily satisfies an older `go.mod` directive) and the Docker build succeeds against the corrected base image.

**Why it matters**: `go mod init`'s auto-detected version is a floor set by whoever's machine happened to run it, not a real requirement — always let `go mod tidy` compute the actual minimum, and make sure any Docker base image (or CI Go version) satisfies that computed minimum, not whatever a contributor's local toolchain happens to be.
