# gh-client-secret

`INSIDEOUT_GH_CLIENT_SECRET` is the GitHub App OAuth client secret. InsideOut
stores it for the future invite-link OAuth consumer; the current server has no
reader for this variable.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-client-secret.key` (manifest
`github/app.yaml`).

## 2. How it is propagated

It is retained in the host key store only. It has no `.env.example` catalog
entry or expected export lane, and the current server does not read it even if
someone exports the name manually.

## 3. How to set it

1. In the GitHub App settings page, generate or reveal the OAuth client
   secret. Keep the value in a password manager while copying it.
2. Run `INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init` to create the
   `0600` placeholder.
3. Use a secure local editor or secret-manager export to create a temporary
   env-format file containing the assignment, then let `keys.py` copy it into
   the placeholder without printing it:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill /path/to/temporary-env
   ```

   Remove the temporary file through the same secure tool. Do not put this
   variable in the repository `.env`, a command argument, or a tracked file;
   no current consumer needs it there.

If the future OAuth integration is not enabled, leave the initialized file
empty. That optional disabled lane is valid and needs no provider value.

## 4. How to test it

This key has **placement-only acceptance** because no current server reader
exists. Run the scoped audit and the per-key shape test:

```bash
INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py audit
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_gh-client-secret.py' -v
```

**Expected result:** the audit reports `key=filled` when the future credential
is retained, or `key=empty` for the disabled lane; the test passes without
printing its value. There is no live OAuth probe to run today.

**If it fails:** recreate the placeholder with `keys.py init`, restore the
secret from the provider or password manager, and confirm the file mode is
`0600`. Do not invent a runtime check or claim that this credential affects the
current login flow.

## 5. How it is used

Nothing reads it today. It is reserved for a future GitHub OAuth invite-link
flow; retaining it avoids an unnecessary provider regeneration later.

## 6. Installation guide

The [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access)
and [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access)
call out its placement-only status. The installation front door is the
[own-server installation path](../../installation.md#your-own-server).
