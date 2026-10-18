# gh-app-private-key

`INSIDEOUT_GH_PRIVATE_KEY` is the RSA private-key PEM downloaded from the
GitHub App settings page. It is the authenticating half of the App
credentials.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-app-private-key.key` (manifest
`github/app.yaml`). The canonical store copy is a raw, multi-line PEM with
mode `0600`.

## 2. How it is propagated

The server accepts either an inline environment value with literal `\n`
escapes or the preferred `_FILE` pointer. `INSIDEOUT_GH_PRIVATE_KEY_FILE`
wins when set and reads the target file raw. Keep one lane active: inline PEM
or `_FILE`, never both.

## 3. How to set it

The safest lane is the file form:

1. In GitHub App settings, choose **Generate a private key** and download the
   PEM. Keep the downloaded file local; do not paste it into a command line.
2. Copy that raw file into
   `hosts/<host>/keys/github/.secrets/gh-app-private-key.key` with mode `0600`
   (a file manager or an equivalent `install -m 600` command is fine).
3. Run `./scripts/env.sh edit`, set
   `INSIDEOUT_GH_PRIVATE_KEY_FILE` to the absolute target path, and leave
   `INSIDEOUT_GH_PRIVATE_KEY` empty.
4. Run the host `keys.py init` and `fill` commands from
   [gh-app-id](gh-app-id.md). `fill` records the pointer; it must not rewrite
   the raw PEM.

If an inline deployment is required, enter the PEM with literal `\n` escapes
through `env.sh edit` and leave `_FILE` empty. Keep the store's canonical copy
raw and multi-line; copying an escaped string into the PEM `.key` file fails
the shape test and the file reader. If the GitHub App integration is disabled,
leave both PEM lanes blank and keep the initialized placeholders.

## 4. How to test it

Use the signed private-repository delivery walk in
[gh-app-id](gh-app-id.md): redeliver a `push` or `pull_request` event from the
GitHub App delivery page and require HTTP 200 plus matching guide evidence.
When the integration is disabled, run
`INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py audit` and accept
`key=empty` for this optional kind instead.

**Expected result:** the server starts without a `private key unusable`
warning, and a delivery with an installation id can fetch the private guide.
With the integration disabled, an empty optional placeholder is expected and no
private-guide delivery is attempted.

**If it fails:** check the PEM has both BEGIN and END lines, the App ID is from
the same App, `_FILE` points to the raw PEM, and the App is installed on the
repository. Do not fix this by printing the PEM or adding it to a shell
argument. The automated placement check is
`cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_gh-app-private-key.py' -v`.

## 5. How it is used

The server signs an RS256 App JWT, exchanges it for a short-lived installation
access token, and uses that token to fetch private `insideout.yaml` guide
content for matching GitHub deliveries.

## 6. Installation guide

Follow the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [own-server installation path](../../installation.md#your-own-server).
