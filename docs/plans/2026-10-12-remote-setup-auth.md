# Remote setup authentication staging (2026-10-12)

Goal: make `insideout setup` collect remote credentials in the order the
user needs them: collect the non-secret setup fields, probe the API, request
an emailed code or magic link when selected, then collect the masked value and
complete the session. Password login follows the same post-probe prompt.

This is the first follow-up slice to the completed setup TUI. It deliberately
does not change refresh-token persistence, installer dispatch, local setup, or
MCP registration; those remain separate slices so each change has one
end-to-end proof.

## Checklist

- [x] Add a failing flow test proving code and magic-link prompts happen after
      their request succeeds.
- [x] Move secret collection behind an injected prompt callback and keep the
      pure flow usable with pre-filled secrets in headless tests.
- [x] Split the TUI remote form so the masked prompt runs after the request.
- [x] Run targeted Go tests, the relevant package suite, and a PTY smoke walk
      with a local fake API or the documented hosted probe.
- [x] Review the diff for secret handling, error ordering, and unchanged local
      behavior.
- [x] Record the accepted slice in a dated changelog and update this plan and
      the task-board row.

Slice closed 2026-10-12. Follow-up work stays in separate plans so the next
change can add refresh-token persistence without changing this authentication
ordering contract.

## Verification contract

The decisive behavior is observable at the client boundary: for code and
magic-link methods, the request method must be called before the secret prompt,
and the verification/consume method must receive the prompted value. Password
login must still probe before prompting and call login once. Empty or failed
prompt results must not persist a config file.
