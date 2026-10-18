# mail-dev-body

`INSIDEOUT_MAIL_DEV_BODY` controls whether mail-dev mode includes rendered
message bodies in local server logs. Blank means the safe default **off**.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-dev-body.key` (manifest
`mail/dev.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into the mail sender at boot. It is
only consulted when `INSIDEOUT_MAIL_API_KEY` is empty; a real provider key sends
mail instead of using this log-only branch.

## 3. How to set it

1. For a disposable local run only, run `./scripts/env.sh edit` and set
   `INSIDEOUT_MAIL_DEV_BODY` to exactly `1`. Leave it blank everywhere shared.
2. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

Never enable this setting on a deployed host: rendered bodies contain login and
reset tokens.

## 4. How to test it

Run the shape test. If a local disposable login is needed, inspect only the
metadata line `mail dev mode — not sent`; do not copy or print the rendered body
because it contains a usable token. Remove the local log after the walk.

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_mail-dev-body.py' -v
```

**Expected result:** the test accepts only blank or `1`; a local dev-mode send
logs `mail dev mode — not sent`, and setting `1` adds the body only to that
local log.

**If it fails:** clear the value on shared hosts, or set it exactly to `1` in a
disposable local environment with the API key empty. Restart after editing so
`MailDevBody` is reloaded.

## 5. How it is used

The mail sender reads the switch once at boot. It never changes provider
requests and has no effect when real mail is enabled.

## 6. Installation guide

Use the [guided mail setup](../../key-setup-tutorial.md#5-configure-mail-when-auth-email-must-leave-the-server),
the [mail testing tutorial](../../key-testing-tutorial.md#6-prove-mail-delivery),
and the [own-server installation path](../../installation.md#your-own-server).
