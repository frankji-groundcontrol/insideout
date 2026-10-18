# Mail key (Resend)

`INSIDEOUT_MAIL_API_KEY` enables transactional auth email for magic-link login,
one-time codes, password reset, and invitations.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-api-key.key` (manifest
`mail/resend.yaml`).

## 2. How it is propagated

The value moves from `.env` through `dev.sh` into the mail sender. A real key
requires both [mail-from](mail-from.md) and
[public-url](public-url.md); those companions determine sender acceptance and
where links point.

## 3. How to set it

1. Create a send-scoped API key in the Resend dashboard. Keep it in a password
   manager while copying it; never place it in a command argument.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_MAIL_API_KEY`, and enter the
   key at the hidden prompt. Set the verified sender and public URL in the same
   session.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

An empty value is a valid local mail-dev lane: messages are logged as not sent.
It is not a delivery configuration.

## 4. How to test it

Run the real provider leg of the smoke flow with an operator-owned, verified
recipient. Run it from the repository root with **no `SMOKE_BASE`** so the
script boots its own server against an isolated disposable database. Before
starting, the database must be migrated, `signup_mode=open` must already be
set for that disposable database, and `DATABASE_OWNER_URL` must be available
so the sanctioned `promote-admin` step can grant the smoke account its probe
permission. The run creates throwaway users and rows and leaves the promoted
smoke account behind for cleanup; it must never target a hosted or shared
database. It never prints the API key:

```bash
MAIL_PROBE_RECIPIENT='replace-with-verified-recipient'
SMOKE_MAIL_TO="$MAIL_PROBE_RECIPIENT" ./server/scripts/smoke.sh
unset MAIL_PROBE_RECIPIENT
```

**Expected result:** the output reports `provider accepted the send
(status=sent)` and the recipient receives the probe message. Redeem one
invitation, magic link, or login code from that inbox for the end-to-end proof.
An auth request returning HTTP 200 by itself is only the anti-enumeration
contract, not delivery evidence.

**If it fails:** `401`/`403` means the key is invalid or revoked; a sender or
recipient refusal means the domain, From address, or recipient policy is wrong;
a missing public URL prevents a real-mail boot. Correct the named companion or
rotate the provider key, then rerun the probe without copying provider bodies
into logs.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_mail-api-key.py' -v
```

## 5. How it is used

The sender adds a Bearer credential to the provider's email API. Auth flows
intentionally keep their HTTP response uniform; the admin provider probe is
the response-truthful delivery signal.

## 6. Installation guide

Follow the [guided mail setup](../../key-setup-tutorial.md#5-configure-mail-when-auth-email-must-leave-the-server),
the [mail testing tutorial](../../key-testing-tutorial.md#6-prove-mail-delivery),
and the [own-server installation path](../../installation.md#your-own-server).
