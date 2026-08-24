# Interactive CLI prompts must share one buffered stdin reader

**Learned** 2026-08-24 from the auth-parity slice
([changelog](../changelogs/2026-08-24-auth-parity-cli-mcp.md)) while
live-exercising `insideout reset-password`.

## The lesson

`prompt()` built `bufio.NewReader(os.Stdin)` per call. The first prompt's
reader buffers **all** of stdin into memory, returns line one, and is
discarded with its buffer; the second prompt's fresh reader finds stdin
already drained and returns empty. Symptom: a two-prompt command fed
`printf 'link\nnewpassword\n'` answers "link and password are required"
— while each prompt alone works fine, so unit tests of the helpers pass
and only the live two-prompt path fails.

Every prompt in one command run must read through one shared reader
(package-level `var stdin = bufio.NewReader(os.Stdin)` in `cli_auth.go`).

## When to apply

Any CLI (or test harness) that reads stdin more than once: prompts,
confirmations, piped multi-line input. Trust signal: the Nth prompt
returns empty although input was piped, and the first N−1 worked.

## Evidence

`reset-password` failed live with both lines piped; after switching all
prompts (and `credentials()`) to the shared reader, the same pipe drove
the full reset → old-password-401 → new-password-login sequence green.
