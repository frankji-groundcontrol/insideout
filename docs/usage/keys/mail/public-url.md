# public-url

`INSIDEOUT_PUBLIC_URL` is the browser origin used as the base of every emailed
invite, magic-link, and password-reset URL.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/public-url.key` (manifest
`mail/resend.yaml`).

## 2. How it is propagated

The value moves from `.env` through `dev.sh` into auth-mail link construction.
It is a URL origin (`scheme + host`, with no trailing path).

## 3. How to set it

1. Choose the public origin that recipients can open, such as the deployed
   app origin. Local mail-dev runs may keep the catalog localhost default.
2. For a real mail key, run `./scripts/env.sh edit` and set
   `INSIDEOUT_PUBLIC_URL` explicitly; do not rely on an unset environment
   variable or a random Flutter web port.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

## 4. How to test it

Run `./scripts/env.sh check server` for the shape gate, then boot through
`./scripts/dev.sh -C server ...` or the guided setup. The config loader must
accept the mail key/public URL pair. Run the real mail probe and inspect the
delivered link without copying its token; its host must equal this origin.

**Expected result:** the server accepts the mail configuration and every
invite/magic/reset link opens at the configured public origin, not localhost.

**If it fails:** a missing explicit URL with a mail key causes config startup
failure; a wrong host in a delivered link means the running process loaded an
old `.env` or the deployment variable was not updated. Set the origin, restart,
and repeat the provider probe.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_public-url.py' -v
```

## 5. How it is used

Auth flows prefix this origin when rendering browser links. It is also the
reference origin for the browser cookie and CORS choices.

## 6. Installation guide

Use the [guided mail setup](../../key-setup-tutorial.md#5-configure-mail-when-auth-email-must-leave-the-server),
the [mail testing tutorial](../../key-testing-tutorial.md#6-prove-mail-delivery),
and the [own-server installation path](../../installation.md#your-own-server).
