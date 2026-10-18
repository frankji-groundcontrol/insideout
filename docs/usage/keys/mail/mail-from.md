# mail-from

`INSIDEOUT_MAIL_FROM` is the From header used by every auth email. It must be a
sender or domain verified by the configured mail provider.

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/mail-from.key` (manifest
`mail/resend.yaml`). The value is a single line such as `Name <addr@domain>`.

## 2. How it is propagated

The value moves from `.env` through `dev.sh` into the mail sender. The provider
uses it for magic-link, code, reset, and invitation messages.

## 3. How to set it

1. Verify the sending domain or address in the Resend dashboard. Use a sender
   that the API key is allowed to send from; the sandbox sender is restricted
   to the provider account's own address.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_MAIL_FROM`, and enter the
   display-name sender at the hidden prompt.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

## 4. How to test it

Run the real mail probe from [mail-api-key](mail-api-key.md), then inspect the
provider-accepted message in the verified recipient's inbox.

**Expected result:** the probe reports `status=sent`, the message arrives, and
its From header matches the configured display name and verified address.

**If it fails:** a provider refusal naming the sender means the domain/address
is unverified or the key is scoped to another sender. Correct verification or
the sender value and repeat the probe. Do not diagnose delivery from an auth
endpoint's uniform HTTP 200.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_mail-from.py' -v
```

## 5. How it is used

The value is serialized as the provider `from` field for every transactional
auth message.

## 6. Installation guide

Follow the [guided mail setup](../../key-setup-tutorial.md#5-configure-mail-when-auth-email-must-leave-the-server),
the [mail testing tutorial](../../key-testing-tutorial.md#6-prove-mail-delivery),
and the [own-server installation path](../../installation.md#your-own-server).
