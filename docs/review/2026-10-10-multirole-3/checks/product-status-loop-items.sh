#!/bin/sh
# TEST-08 check — PRODUCT.md's status section must enumerate the
# unimplemented loop items (Product Branch/Merge, Decision Log, scoped
# feedback invitations) exactly as it already discloses the roles gap.
# Red while the status paragraph disclaims only the roles gap.
set -eu
cd "$(dirname "$0")/../../../.."
[ -f PRODUCT.md ] || { echo "FAIL: PRODUCT.md not found"; exit 1; }

status=$(sed -n '/^## Superseded assumptions and current status/,/^## Brand commitments/p' PRODUCT.md)
for item in "Product Branch" "Decision Log" "feedback invitation"; do
  printf '%s\n' "$status" | grep -qi "$item" ||
    { echo "FAIL: PRODUCT.md's status section does not disclose the unimplemented '$item' loop item"; exit 1; }
done

echo "OK: PRODUCT.md's status section enumerates the unimplemented loop items"
