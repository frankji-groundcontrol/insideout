#!/bin/sh
# COMP-01 check — the PRODUCT.md promise register must exist and cover the
# promise sections (PRODUCT.md 64-390) with a per-promise IMPLEMENTED /
# PARTIAL / DEAD / DOCS-ONLY stamp. Red on a tree where lead 14 was never
# run, which is review 3's finding.
set -eu
cd "$(dirname "$0")/../../../.."
REG=docs/review/2026-10-10-multirole-3/50-promise-register.md

[ -f "$REG" ] || { echo "FAIL: $REG missing — the lead-14 register was never produced"; exit 1; }

# Every PRODUCT.md promise section inside the walked range must appear.
for section in "Canonical journey" "first workable version" "One PRD core" \
               "Facts, evidence, and privacy" "Product version control" \
               "Collaboration and authority" "shared Roadmap Graph" \
               "Git, CLI, MCP, and Agents" "Completion and learning" \
               "end-state loop"; do
  grep -q "$section" "$REG" || { echo "FAIL: register does not walk PRODUCT.md section '$section'"; exit 1; }
done

# The review's seeded gap promises must each be classified (not inherited).
for promise in autosave "Product Branch" "Decision Log" "feedback invitation" \
               persona lens "dependenc" "per-Project role" "facts surface"; do
  grep -qi "$promise" "$REG" || { echo "FAIL: register never classifies the promise '$promise'"; exit 1; }
done

# The confirmed-implemented set must be classified too — a gap-only list is
# not the classification the feature-implementation brief demanded.
for promise in "audience" "Commit" "roadmaptime" "readiness" "evidence kind"; do
  grep -qi "$promise" "$REG" || { echo "FAIL: register never classifies the promise '$promise'"; exit 1; }
done

# Every row must carry a stamp and a PRODUCT.md anchor.
stamped=$(grep -c 'IMPLEMENTED\|PARTIAL\|DEAD\|DOCS-ONLY' "$REG" || true)
[ "$stamped" -ge 40 ] || { echo "FAIL: only $stamped stamped lines — the walk is not per-promise"; exit 1; }
anchors=$(grep -c 'PRODUCT\.md:' "$REG" || true)
[ "$anchors" -ge 40 ] || { echo "FAIL: only $anchors lines cite PRODUCT.md — rows are not promise-anchored"; exit 1; }

# The five overlapping review gap-lists must be reconciled by name.
for finding in "FULL-03" "CQ-02" "SEC-04" "TEST-08" "architect"; do
  grep -q "$finding" "$REG" || { echo "FAIL: register does not reconcile the '$finding' gap-list"; exit 1; }
done

echo "OK: $REG walks the promise sections with per-promise stamps and reconciles the five gap-lists"
