#!/usr/bin/env bash
# Dump the insideout schema from DATABASE_URL to a dated file under
# ./backups/ (never committed — user data stays on the operator's disk).
# CC-03 (review 2): this is the portable half of the backup story; the
# managed half (Supabase automated backups) is the operator's toggle.
#
# Usage:  ./scripts/backup-db.sh [output-dir]   (default: ./backups)
# Env:    DATABASE_OWNER_URL — the owner-role DSN. The app role is an
#         RLS subject and pg_dump fails against it by design ("query
#         would be affected by row-level security policy"); the owner
#         role bypasses RLS and exists for exactly this kind of
#         schema-level operation.
#
# The dump is schema+data of the insideout schema only, plain SQL, so a
# restore works on any Postgres with the two roles present (see
# docs/usage/backup-restore.md for the full runbook). It holds credential
# hashes, so the file (and a new output directory) is owner-only.
set -euo pipefail

# pg_dump creates its file under the caller's umask (0644 under the usual 022).
umask 077

out_dir="${1:-backups}"
: "${DATABASE_OWNER_URL:?DATABASE_OWNER_URL must be set — source it from your env, never hardcode it here}"

mkdir -p "$out_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
file="$out_dir/insideout-$stamp.sql"

# -n insideout      : only the product schema (roles live outside it)
# --no-owner        : the dump restores under whoever runs it
# --clean --if-exists: restore can target a non-empty schema
pg_dump "$DATABASE_OWNER_URL" \
  --schema=insideout \
  --no-owner \
  --clean --if-exists \
  --file="$file"

# umask covers a fresh file; a same-second leftover keeps its old, wider mode.
# A failed chmod aborts under set -e rather than leaving a readable dump.
chmod 600 "$file"

# Print the path and size only — never the DSN or row counts.
echo "wrote $file ($(du -h "$file" | cut -f1))"
