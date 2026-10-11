# A send-scoped API key's 401 looks exactly like a dead key

## What was learned

Resend (like most providers with scoped keys) answers a **restricted** key
with `401 restricted_api_key` on endpoints outside its scope — which is
byte-for-byte the failure shape of a **revoked** key on those same
endpoints. Testing a send-only key against `GET /domains` concludes "the
key is dead" when the key is perfectly healthy for its purpose. The valid
liveness test for a scoped credential is the capability it exists for: for
a mail key, **one real send** — and the send response distinguishes
auth-failure (401) from policy (403 validation) from success (200 + email
id) far more precisely than any read endpoint.

Three provider-side facts that only the events surface shows:

- `sent` (handed to the receiving server), `delivered`, `bounced`, and
  `suppressed` are four different truths; a send can 200 and still be
  suppressed recipient-side.
- An address that bounced once gets **suppressed** on later sends —
  suppressed mail never arrives and never errors.
- The account's own address always delivers under the sandbox sender, so
  it is the calibration point when testing.

## Evidence

The 2026-10-11 re-key
([changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)): the
owner's fresh key returned 401 on `/domains` — indistinguishable from the
revoked key it replaced until a direct `POST /emails` answered 200 with an
id. The mailed-flow pass then read every email back from the provider's
API (a send-scoped key can still read its own sends) and redeemed each
token; the earlier rehearsal address showed both `bounced` and later
`suppressed` for exactly this reason.

## Scope

Judge scoped credentials by their granted capability, never by an
out-of-scope endpoint; and when verifying a mail funnel, treat the
provider's event stream (sent/delivered/bounced/suppressed) as part of the
contract — the HTTP layer alone cannot see a suppressed recipient.
