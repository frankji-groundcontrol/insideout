# A lockfile diff confined to SDK-pinned packages is toolchain drift, not intent

## What was learned

When `client/pubspec.lock` differs between two hosts and the changed
packages are exactly the ones the Flutter SDK dictates (`test_api` and
`matcher` pinned exactly by `flutter_test`; `meta`, `vector_math`, and `intl`
constrained by `flutter` and `flutter_localizations`), the diff records which
SDK last ran `flutter pub get`. It is not a dependency decision. Reverting it
because it looks accidental, or committing it because it is in the tree,
are both guesses; the evidence is the SDK's own pubspecs.

## How to apply

1. List what moved: `git diff -- client/pubspec.lock`.
2. Read those packages' pins in the SDK each host runs
   (`packages/flutter_test/pubspec.yaml` and its siblings) and the SDK the
   repo pins (CI, Dockerfile). A diff equal to one SDK's pins is drift.
3. Decide by the repo's pinned SDK, not by whichever host wrote last: commit
   the lock that satisfies `flutter pub get --enforce-lockfile` under the pin,
   as its own commit so it reverts alone.
4. Fix the cause — the host's SDK — or the diff returns on the next
   `pub get`.

## Evidence

2026-10-09 sync: five packages moved. The working-tree lock equalled Flutter
3.44.8's pins, the committed lock matched 3.47.0's, and the committed lock
failed `--enforce-lockfile` under 3.44.8
([changelog](../changelogs/2026-10-09-host-sync.md),
[issue](../issues/2026-10-09-host-setup-drift-between-macs.md)).
