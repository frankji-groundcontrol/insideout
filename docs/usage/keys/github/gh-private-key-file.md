# gh-private-key-file

`INSIDEOUT_GH_PRIVATE_KEY_FILE` points at the canonical raw PEM for the
GitHub App. It is the config companion of
[gh-app-private-key](gh-app-private-key.md), not a second private key.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-private-key-file.key` (manifest
`github/app.yaml`). Its value is a path ending in
`gh-app-private-key.key` in the same host store.

## 2. How it is propagated

The path moves through `.env` and `dev.sh` into `github.LoadPrivateKey`.
When it is non-empty, the server reads the target raw and ignores the escaped
inline `INSIDEOUT_GH_PRIVATE_KEY` value.

## 3. How to set it

1. Complete the raw PEM setup in
   [gh-app-private-key](gh-app-private-key.md).
2. Run `./scripts/env.sh edit` and set
   `INSIDEOUT_GH_PRIVATE_KEY_FILE` to that PEM's absolute path. Leave the
   inline PEM variable empty.
3. Fill the pointer into the host store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

The pointer is safe to record; the PEM itself must never appear in command
arguments or tracked files. Leave this optional pointer blank when the inline
PEM lane is active or the GitHub App integration is disabled.

## 4. How to test it

Run the per-key pointer test and then redeliver a signed private-repository
GitHub App event as described in [gh-app-id](gh-app-id.md).
For an inline or disabled lane, the same test accepts an empty optional
pointer; confirm that state with
`INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py audit`.

**Expected result:** the pointer test confirms the target exists, the server
starts without a private-key warning, and a matching delivery returns HTTP 200
with guide evidence when the repository contains a matching guide. An empty
pointer is expected for the inline or disabled lane.

**If it fails:** use an absolute path, check that the target is the raw
multi-line PEM, leave the inline variable empty, and verify mode `0600` on both
files. Run:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_gh-private-key-file.py' -v
```

## 5. How it is used

It selects the raw-file lane in `github.LoadPrivateKey`, preventing escaped
environment PEM text from reaching the file reader.

## 6. Installation guide

Use the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [own-server installation path](../../installation.md#your-own-server).
