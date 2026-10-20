# 2026-10-20 — Adopt the review-4 recommended settings

The owner asked to close the remaining review-4 gaps with the settings
the records already recommend. Operator consoles (hosted role, Railway
variables, GitHub App events, key rotation) stay the owner's hands.
History is not rewritten.

## Adopted

| Gap | Setting |
| --- | --- |
| Retention | 365 days after the last message. Tombstone erasure. `DELETE /me` after the password, human-only. No export requirement. No response-time promise. |
| History | Accept the existing authors. Do not rewrite. |
| Migration cutoff | Leave `DEPLOYED_THROUGH` at `20261006120000`. Do not guess the hosted table. |
| Invitation route | "The server's administrator." No mailbox and no new form. |
| Hand-over | A non-admin owner cannot give a project away. An omitted `ownerId` keeps the owner. The last workspace admin cannot be removed or demoted. |
| Session swap | While a login is held, do not advertise `auth_login_code_verify`, `auth_magic_link_consume`, or `auth_logout`. |
| Completion | `in_progress` to `done` needs linked evidence or a one-sentence reason. |
| Version numbers | While the major is 0, a break of a documented contract raises MINOR; anything else raises PATCH. Every tag stays a normal release. |
| Kept as built | Agent tokens never decide a proposal. Critic skip may finalize and is announced. Register keeps its distinct 409. Deadlines stay noon UTC. One repo links to one project. No release-variable retarget. Hosts stay tracked as scrubbed files. |

## Checklist

- [x] Retention and erasure — on local `main` as `d5a45fe` and `3ecef69` (from `77eeed7`, `9ec7b31`).
- [ ] Completion rule — stopped. Red `65569dc` is `2986f4b`. Fix `1cb3d1d` conflicted in five doc indexes and was aborted.
- [x] Session-tool withhold — on local `main` as `8f3a76d` and `9e6e68e` (from `59a57c3`, `e235c15`).
- [ ] Omitted owner and last admin — skipped. Verdict REJECTED (`c2ee18e`, `221fb87`): the diff left the handover files.
- [ ] Decision record for the keep-as-built rows — skipped. Verdict REJECTED (`17f15e7`): the record does not state only these settings.
