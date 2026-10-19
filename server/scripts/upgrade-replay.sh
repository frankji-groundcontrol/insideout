#!/usr/bin/env bash
# TEST-01 (review-3): the migration upgrade path replays real old data before
# production ever sees it. Every other lane migrates a fresh empty database,
# so the pending migrations first meet real rows on the hosted database. This
# script, on a fresh scratch database:
#   1. replays the migration files the hosted stack already has (everything
#      through DEPLOYED_THROUGH below) with the same bookkeeping Store.Migrate
#      uses — the same tracking table, one transaction per file, the owner
#      role — so the server's runner afterwards sees a genuinely old schema,
#   2. loads the pinned old-shape seed (upgrade-seed.sql),
#   3. migrates the rest of the way with the server's own runner
#      (`go run ./cmd/insideout migrate`),
#   4. asserts the old data met the pending migrations correctly: the legacy
#      short invite code rotated, the modern one untouched, every seeded row
#      (including one authored by a removed member) still present, every
#      migration file recorded.
# ci.yml's upgrade-replay job runs smoke.sh against the result.
#
# Needs DATABASE_URL, DATABASE_OWNER_URL and INSIDEOUT_JWT_SECRET in the
# environment (CI exports them; locally export the scratch cluster's set).
# It refuses to start when an insideout schema already exists: a replay must
# begin empty, or it is not a replay.
#
# Usage:
#   cd server && ./scripts/upgrade-replay.sh
set -euo pipefail

cd "$(dirname "$0")/.." # run from server/

# The deployed production set as of 2026-10-08: the last migration the hosted
# database already has, so everything after it is what this lane exercises
# before a deploy does. Bump it when production catches up.
#
# OPEN OWNER QUESTION (rv4-TEST-07, rv4-CLOSE-09): docs/HANDOFF.md says the
# hosted database was migrated through 20261011140000 on 2026-10-11, so this pin
# is probably stale, but only the hosted database knows (SELECT max(filename)
# FROM insideout.schema_migrations). Bumping it alone turns this lane red: the
# seed and the rotation assertions below are aimed at 20261010110000, so the
# same change must re-aim them at the migrations that follow the confirmed cutoff.
DEPLOYED_THROUGH=20261006120000

# The two invite-code shapes pinned in upgrade-seed.sql; the rotation
# assertions below turn on them.
LEGACY_CODE=448172
MODERN_CODE=9f8e7d6c5b4a3021f0e1d2c3b4a59687

: "${DATABASE_URL:?upgrade-replay needs DATABASE_URL (the scratch database)}"
: "${DATABASE_OWNER_URL:?upgrade-replay needs DATABASE_OWNER_URL (insideout_owner; migrations are DDL)}"
: "${INSIDEOUT_JWT_SECRET:?upgrade-replay needs INSIDEOUT_JWT_SECRET (the migrate command loads the server config)}"

# Starting on a database that already has the schema would at best duplicate
# the replay and at worst seed over live-shaped rows.
if [ "$(psql "$DATABASE_OWNER_URL" -tAc \
  "SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'insideout')")" = "t" ]; then
  echo "refusing: an insideout schema already exists — drop it (DROP SCHEMA insideout CASCADE) and rerun on a fresh scratch database" >&2
  exit 2
fi

# Filename prefixes are fixed-width timestamps; this both finds the deployed
# set's last file and fails fast on a typo'd cutoff.
ls db/migrations/ | grep -q "^${DEPLOYED_THROUGH}_" \
  || { echo "cutoff migration ${DEPLOYED_THROUGH}_*.sql not found under db/migrations/" >&2; exit 1; }

echo "== 1/4 replaying the deployed migrations (through ${DEPLOYED_THROUGH}) =="
# The same bootstrap Store.Migrate performs: the schema (so no replayed file
# ever needs database-level CREATE) and the tracking table, as the owner role.
psql "$DATABASE_OWNER_URL" -v ON_ERROR_STOP=1 -q \
  -c "CREATE SCHEMA insideout AUTHORIZATION insideout_owner" \
  -c "CREATE TABLE IF NOT EXISTS insideout.schema_migrations (
        filename text PRIMARY KEY,
        applied_at timestamptz NOT NULL DEFAULT now())"

replayed=0
for file in db/migrations/*.sql; do # glob expansion is filename order
  # Store.Migrate records the full filename incl. .sql; match it exactly or
  # the runner re-applies every replayed file.
  base="$(basename "$file")"
  # Fixed-width prefixes make this a numeric comparison; stop at the first
  # pending (post-cutoff) file.
  if [ "${base:0:14}" \> "$DEPLOYED_THROUGH" ]; then break; fi
  # One transaction per file plus its tracking row, as insideout_owner —
  # the shape Store.Migrate runs.
  psql "$DATABASE_OWNER_URL" -v ON_ERROR_STOP=1 -1 -q -f "$file" \
    -c "INSERT INTO insideout.schema_migrations (filename) VALUES ('$base')"
  replayed=$((replayed + 1))
done
total="$(ls db/migrations/*.sql | wc -l | tr -d ' ')"
[ "$replayed" -lt "$total" ] \
  || { echo "nothing pending after ${DEPLOYED_THROUGH} — the cutoff must stay behind HEAD or this lane replays nothing" >&2; exit 1; }
echo "   replayed $replayed of $total migration files"

echo "== 2/4 loading the pinned old-shape seed (scripts/upgrade-seed.sql) =="
psql "$DATABASE_OWNER_URL" -v ON_ERROR_STOP=1 -1 -q -f scripts/upgrade-seed.sql

echo "== 3/4 migrating the pending set to HEAD (the server's own runner) =="
go run ./cmd/insideout migrate

echo "== 4/4 asserting the old data met the pending migrations =="
q() { psql "$DATABASE_OWNER_URL" -tAc "$1"; }
assert_eq() { # assert_eq GOT WANT LABEL
  if [ "$1" = "$2" ]; then
    echo "   ok: $3"
  else
    echo "   LANE FAILURE: $3 — got '$1', want '$2'" >&2
    exit 1
  fi
}

# 20261010110000 must meet the seeded legacy short code here, not in
# production. The sub-32 count is the lane's tripwire: replaying past the
# cutoff (or seeding after HEAD) leaves it at 1.
assert_eq "$(q "SELECT count(*) FROM insideout.workspaces WHERE length(code) < 32")" \
  0 "no workspace keeps a sub-32 invite code (the legacy one was rotated)"
assert_eq "$(q "SELECT count(*) FROM insideout.workspaces WHERE code = '$MODERN_CODE'")" \
  1 "the already-modern invite code was left untouched"
# users|workspaces|ideas|prds|project_updates|updates-by-the-removed-member
assert_eq "$(q "SELECT (SELECT count(*) FROM insideout.users)
                 || '|' || (SELECT count(*) FROM insideout.workspaces)
                 || '|' || (SELECT count(*) FROM insideout.ideas)
                 || '|' || (SELECT count(*) FROM insideout.prds)
                 || '|' || (SELECT count(*) FROM insideout.project_updates)
                 || '|' || (SELECT count(*) FROM insideout.project_updates pu
                             JOIN insideout.users u ON u.id = pu.author_id
                            WHERE u.email = 'seed-removed@example.test')")" \
  "2|2|2|2|2|1" "every seeded row survived, incl. one update authored by a removed member"
assert_eq "$(q "SELECT count(*) FROM insideout.schema_migrations")" \
  "$total" "every migration file is recorded (replay + HEAD pass cover them all)"

echo "UPGRADE REPLAY GREEN: $replayed deployed files replayed, $((total - replayed)) pending migrations met the seeded data"
