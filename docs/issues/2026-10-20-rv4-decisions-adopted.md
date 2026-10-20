# Review 4 decisions adopted

Recorded 2026-10-20. These rows were open owner decisions in
[the open findings](2026-10-18-review-4-open-findings.md). Each is now
decided. The Decision cell is the whole decision; this record adds no
further policy. The release sentences are also the runbook in
[releasing](../usage/releasing.md).

Rows not named below are unchanged. Operator consoles at the end were not
decided.

## Decided

| Register ID | Decision |
|---|---|
| rv4-PUB-06 | history authors are accepted and history is not rewritten |
| rv4-TEST-07, rv4-CLOSE-09 | DEPLOYED_THROUGH stays 20261006120000 because the hosted table was not queried |
| rv4-USER-12 | an invitation is requested from the server's administrator, with no mailbox and no new form |
| rv4-TEST-03 | a non-admin owner cannot give a project away; an omitted ownerId keeps the owner; the last admin cannot be removed or demoted |
| rv4-FEAT-01 | agent tokens never decide a proposal |
| rv4-ARCH-05 | the release workflow reads no API-base variable |
| rv4-AIDEV-02 | a skipped critic may reach finalize and is announced |
| rv4-SEC-03 | POST /auth/register keeps its distinct 409 |
| rv4-COMP-02 | deadlines stay noon UTC of the picked day |
| rv4-COMP-05 | one repository links to one project and the first successful link wins |
| rv4-PUB-10 | hosts/ stays tracked with scrubbed paths |
| rv4-PUB-12 | the public host stays in dated records |
| rv4-PM-05 | while the major is 0, a break of a documented command, flag, config key, or API contract raises MINOR and anything else raises PATCH |
| rv4-PUB-03, rv4-PUB-07, rv4-PM-05 | every tag remains a normal release |
| rv4-PM-05 | a bad release is superseded, not deleted |
| rv4-PM-05, rv4-TEST-01 | the release job keeps waiting for a green ci run and does not gain Postgres |
| rv4-PUB-03 | installs without gh stay on the checksum |

The deadline sentence does not decide whether timestamps show in the
viewer's zone. The last workspace admin cannot be removed or demoted:
`memberships.go` already returns `ErrLastAdmin`. What makes `v1.0.0` was
not decided.

## Still the owner's

| Console | Register ID |
|---|---|
| hosted role check | rv4-DATA-04, rv4-DATA-03 |
| INSIDEOUT_TRUSTED_PROXY_HOPS=1 | rv4-SEC-02 |
| GitHub deployment_status | rv4-FEAT-07 |
| key rotation | rv4-PUB-01, rv4-PUB-05 |
| the Railway VERSION variable | rv4-PM-05 (settings) |
