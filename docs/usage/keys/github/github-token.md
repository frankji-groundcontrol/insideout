# GitHub token (optional PAT)

`GITHUB_TOKEN` is an optional personal access token (PAT) that only
`insideout setup --check` reads: it validates the token against the GitHub API.
The server never sends it. Commit sync (`insideout sync` and the push webhook)
reads a private repository with the GitHub App's installation token and a
public one anonymously (rv4-FEAT-04), so a PAT no longer raises its rate limit.
Whether to retire the key is residue R3-27 in the
[open findings](../../../issues/2026-10-18-review-4-open-findings.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/github-token.key` (manifest
`github/tokens.yaml`). An empty `0600` file is the intended optional state.

## 2. How it is propagated

`dev.sh` exports it with the rest of `.env`, but `docker-compose.yml` does not
pass it to the container and the server reads it nowhere, so a token in the
server's environment changes nothing. The commit sync takes its credential from
the GitHub App ([gh-app-id.md](gh-app-id.md)): a private repository needs the
App installed on it, a public one syncs anonymously. Only
`insideout setup --check` reads the token, from the environment or the
repository's env file, and calls the GitHub API to validate it. It is not used
by webhook signature verification or App installation-token minting.

## 3. How to set it

1. In GitHub token settings, create a fine-grained token with the narrowest
   read-only repository permission. Keep the expiry and repository scope
   explicit; the token only has to be accepted by the GitHub API.
2. Run `./scripts/env.sh edit`, choose `GITHUB_TOKEN`, and enter the token at
   the hidden prompt.
3. Run the host `keys.py init` and `fill` commands from the GitHub module
   README. An empty value remains valid; nothing in the server needs the token.

## 4. How to test it

Run the status-only check:

```bash
insideout setup --check
```

**Expected result:** the `github` line is `ok`; the command reports only token
metadata and never prints the token. This proves the token only. For evidence
that a private repository syncs, install the GitHub App on it, bind it to a
project and run `insideout sync '<project-id>'`: that exercises the App
credentials ([gh-app-id.md](gh-app-id.md)), not this token.

**If it fails:** HTTP 401/403 means the token is expired, revoked, or missing
the repository permission. A sync rate-limit or not-found response is not about
this token, which the sync does not send; check the App installation on the
repository instead. A public repository syncs with an empty token, so do not
create a PAT solely to make the shape test non-empty.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_github-token.py' -v
```

## 5. How it is used

Only `insideout setup --check` uses it, to confirm the GitHub API accepts it
(status only, the value is never printed). It is not sent on commit sync, does
not grant InsideOut user access and does not replace the App's webhook secret or
private-key pair.

## 6. Installation guide

Use the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [health-check section](../../installation.md#re-verify-any-time) of the
installation guide.
