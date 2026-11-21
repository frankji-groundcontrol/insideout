# Password-reset watch

Date: 2026-10-21.

`POST /api/v1/auth/forgot-password` now returns a `watch` id with its
usual `ok`, except when the send cooldown refuses the request: that
answer carries no id. The cooldown keys on the address and the client,
not on the account, so the missing id reveals nothing. `GET /api/v1/auth/reset-watch/<id>` answers `pending`,
`completed`, or `expired`. Using the reset link, in the browser or in
`insideout reset-password`, marks that watch completed.

`insideout forgot-password` waits on the id. When the link is used
somewhere else, the command drops the saved login and says to sign in
again. An unknown address still receives an id, and that id never
completes, so the command does not reveal who is registered.

The watch rows live in `insideout.reset_watches`
(migration `20261021140000`), outside RLS like `auth_tokens`. The
reaper's retention purge deletes watches that expired more than 30 days
ago.

An emailed sign-in or reset asks for confirmation of the address before
the send. The answer includes `mail`: `accepted` when a provider is
configured, or `logged` when none is. The wizard and
`insideout forgot-password` print that status. The word is the same for
a real address and an unknown one.

## Wave 0 fixes to the slice

Review 5 wave 0 ([plan](../review/2026-11-21-focused-review-5/40-remediation-plan.md)) closed these before the checkpoint commit:

- B (PARA-01): `insideout forgot-password` asks for confirmation only when stdin is a real terminal (`term.IsTerminal` on the fd), so `/dev/null` no longer counts as one.
- C (PARA-08): `ResetWatchStatus` escapes the watch id with `escSeg`, and the escape sweep has rows for `ForgotPasswordWatch`, `RequestLoginCodeStatus`, `RequestMagicLinkStatus` and `ResetWatchStatus`.
- D: the forgot-password answer reads the server clock `s.now()`, not `time.Now`.
- E: the reset mail text says "valid for 60 minutes" again, keeps the reset code, and its HTML footer agrees.
- F (PARA-10): the cooldown answer carries no watch id, and this record and the code comment now say so.
- G (COMP-06): a send the full queue refused answers `accepted`, not `dropped`, because only a real account reaches the queue; `mail.Enqueue` logs the drop on the server.
- H (COMP-04): the retention purge deletes expired `reset_watches` rows; the table is named in the no-RLS lists, and a real-database test pins the purge and the no-RLS table list.
- I (FE-06): the reset page tells a too-short password, a throttle and a network fault apart from an invalid or expired link, in English and Chinese.
