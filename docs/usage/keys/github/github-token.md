# GitHub token (optional PAT)

`GITHUB_TOKEN` is an optional personal access token (PAT) that raises the
GitHub REST rate limit for user-triggered commit sync. Public repositories work
without it; the GitHub App credentials serve the separate signed-webhook guide
path.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/github-token.key` (manifest
`github/tokens.yaml`). An empty `0600` file is the intended optional state.

## 2. How it is propagated

When present, `dev.sh` exports it to the server. `github.FetchCommitsSince`
adds a Bearer header for `insideout sync`; `insideout setup --check` also calls
the GitHub API to validate the token. It is not used by webhook signature
verification or App installation-token minting.

## 3. How to set it

1. In GitHub token settings, create a fine-grained token with the narrowest
   read-only repository permission needed for the projects you bind. Keep the
   expiry and repository scope explicit.
2. Run `./scripts/env.sh edit`, choose `GITHUB_TOKEN`, and enter the token at
   the hidden prompt.
3. Run the host `keys.py init` and `fill` commands from the GitHub module
   README. An empty value remains valid when public repositories and the
   unauthenticated rate limit are enough.

## 4. How to test it

Run the status-only check:

```bash
insideout setup --check
```

**Expected result:** the `github` line is `ok`; the command reports only token
metadata and never prints the token. For capability evidence, bind a private
repository and run `insideout sync '<project-id>'`.

**If it fails:** HTTP 401/403 means the token is expired, revoked, or missing
the repository permission. A sync rate-limit response means the token was not
loaded by the running process; rerun through `scripts/dev.sh` and audit the
host store without printing values. A public repository can still sync with an
empty token, so do not create a PAT solely to make the shape test non-empty.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_github-token.py' -v
```

## 5. How it is used

The token is an optional Bearer credential on GitHub commit-list requests. It
does not grant InsideOut user access and does not replace the App's webhook
secret or private-key pair.

## 6. Installation guide

Use the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [health-check section](../../installation.md#re-verify-any-time) of the
installation guide.
