# Remote setup authentication is staged correctly

`insideout setup` now collects remote secrets in the order the user needs
them. The API is probed before authentication, password input is collected
after that probe, and emailed-code or magic-link input is collected only after
the corresponding request succeeds. Headless callers can still provide a
pre-filled secret through the flow contract.

Verification:

- `go test -C server ./internal/setup/... ./cmd/insideout ./internal/clientconfig ./internal/apiclient -count=1`
- `go vet -C server ./internal/setup/... ./cmd/insideout`
- `git diff --check`
- PTY walks for password and emailed-code login against a local fake API;
  both completed setup, and the password/code values were absent from the
  captured terminal output.

The next setup slices remain separate: refresh-token persistence, early auth
capability gating, installer/TUI unification, local API binding, and MCP
registration.
