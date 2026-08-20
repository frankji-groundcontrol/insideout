# 2026-08-19 — Flutter cutover dropped Ink & Seal

One QA thread on the Flutter client's appearance after the Nuxt → Flutter
hosting cutover. The user rejected treating a client rewrite as a visual
rewrite.

## Entries

### "I saw you totally ditched our visual on nuxt? why?"

**Resolution.** The Nuxt tree (`app/`) was deleted on 2026-08-18 and the
public client is Flutter. The visual ditch was not required by Flutter. It
was an explicit lock in
[`docs/plans/2026-08-17-flutter-client.md`](../plans/2026-08-17-flutter-client.md):
"Use **Material 3**, not Ink & Seal." That lock is reversed. The committed
world stays **国风留白 / Ink & Seal** ([`DESIGN.md`](../../DESIGN.md)).
Flutter remains the client; it must wear the recovered tokens, seal, and
paper chrome.

**Files touched:** the restore plan
[`docs/plans/2026-08-19-restore-ink-seal.md`](../plans/2026-08-19-restore-ink-seal.md),
[`client/lib/theme/`](../../client/lib/theme/),
[`client/lib/app.dart`](../../client/lib/app.dart),
[`client/lib/features/landing/landing_page.dart`](../../client/lib/features/landing/landing_page.dart),
[`client/lib/features/auth/`](../../client/lib/features/auth/),
[`client/assets/seals/yin.webp`](../../client/assets/seals/yin.webp).

### "an infra migration does not mean visual change"

**Resolution.** Agreed. Railway, Go, and Flutter are infrastructure. Celadon
ground, sumi-ink primary action, vermilion seal accent, Song display, and
the AuthDoor paper panel are the product. First restore: tokens + auth/landing
chrome. Assembly landing and roadmap chops stay on the restore plan.

**Files touched:** same as above, plus the Flutter plan correction and the
task board.

### "the motion lost"

**Resolution.** Restored the Nuxt timings: AuthDoor seal stamp (1.6 → 0.88
→ 1.07 → 1), paper panel rise, hero fade/rise, Assembly diagram click-in
(spark / doc / tree pop, then vermilion seal stamp), and in-view step /
close reveals. Same ease `cubic-bezier(0.16, 1, 0.3, 1)`. Reduced-motion
skips the sequence.

**Files touched:**
[`client/lib/theme/ink_motion.dart`](../../client/lib/theme/ink_motion.dart),
[`client/lib/features/landing/`](../../client/lib/features/landing/),
[`client/lib/features/auth/auth_door.dart`](../../client/lib/features/auth/auth_door.dart).

### "the login be a prompt module instead of a redirect? A pop-up prompt modal would probably look a bit better."

**Resolution.** Log in / register on the landing no longer navigate away. They
open the AuthDoor as a prompt overlay (seal stamp, paper panel, Escape and
scrim close) on top of the Assembly page. `/login` and `/register` still exist
for auth-redirect / deep links and render the same landing-plus-prompt.

**Files touched:**
[`client/lib/features/landing/landing_page.dart`](../../client/lib/features/landing/landing_page.dart),
[`client/lib/features/auth/`](../../client/lib/features/auth/).

### "[diagram] The transition animation here is missing."

**Resolution.** Step mini-maps were drawn in the final "you are here" state
with no click-in (that's why later nodes sat ghosted). They now play the
spark → doc → tree → seal sequence up to the current step when they enter
the viewport; remaining nodes stay ghosted. The hero still plays the full
stamp.

**Files touched:**
[`client/lib/features/landing/assembly_diagram.dart`](../../client/lib/features/landing/assembly_diagram.dart),
[`client/lib/features/landing/assembly_step.dart`](../../client/lib/features/landing/assembly_step.dart).
