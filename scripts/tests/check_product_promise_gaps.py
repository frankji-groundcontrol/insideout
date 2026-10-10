#!/usr/bin/env python3
"""Fail when a parked PRODUCT.md promise is not tracked as a dated issue (review-3 wave 4).

Usage: python3 -I scripts/tests/check_product_promise_gaps.py

Every parked slice's premise is derived from the tree first (no
product-branch machinery, no feedback route, no dependency edges, ...).
While a premise holds, the slice must be tracked: a dated record under
docs/issues/, a row for it in docs/issues/README.md, and a link from
PRODUCT.md's status section to the enumeration record. When a slice
ships, its premise disappears and the demand disappears with it
(review-3 CQ-02, FULL-03, SEC-04, USER-04, FEAT-05, FEAT-06).
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]

ENUMERATION = "2026-10-11-product-promise-gaps.md"


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def concat(roots, suffixes):
    parts = []
    for root in roots:
        for path in sorted((ROOT / root).rglob("*")):
            if path.is_file() and path.suffix in suffixes:
                parts.append(path.read_text(encoding="utf-8"))
    return "\n".join(parts)


CODE = concat(["server/internal", "client/lib"], {".go", ".dart"})
CLIENT = concat(["client/lib"], {".dart"})
MIGRATIONS = concat(["server/db/migrations"], {".sql"})
STORE = concat(["server/internal/store"], {".go"})
# Every route the API registers, one HandleFunc line each, so premises can
# ask "is there a route for X?" without importing Go.
ROUTES = "\n".join(
    line for line in concat(["server/internal/api"], {".go"}).splitlines()
    if "HandleFunc(" in line
)


def outward_read_routes_are_auth_only():
    """The audience projections have no unauthenticated (tokenized) read."""
    outward = [l for l in ROUTES.splitlines() if "view" in l or "share" in l]
    return bool(outward) and all("requireAuth" in l for l in outward)


def slice_premises():
    """(record, what it tracks, live?) for every parked slice."""
    return [
        (
            "2026-10-11-product-branch-merge-decision-log.md",
            "Product Branch / Merge and the Decision Log as a surface",
            not re.search(r"product.{0,2}branch|explore direction", CODE, re.I),
        ),
        (
            "2026-10-11-scoped-feedback-invitations.md",
            "scoped external feedback invitations",
            not re.search(r"feedback", ROUTES, re.I),
        ),
        (
            "2026-10-11-tokenized-audience-share-link.md",
            "a tokenized read-only audience-share link",
            outward_read_routes_are_auth_only(),
        ),
        (
            "2026-10-11-persona-grounding-labels.md",
            "persona grounding labels",
            not re.search(r"\bpersonas?\b", STORE, re.I)
            and not re.search(r"grounded", CLIENT, re.I),
        ),
        (
            "2026-10-11-facts-trust-layer.md",
            "the facts trust layer (confirm/correct/remove)",
            "lm.Facts" in read("server/internal/agent/ledger.go")
            and "fact" not in ROUTES.lower(),
        ),
        (
            "2026-10-11-roadmap-lenses.md",
            "per-role Roadmap lenses",
            "enum _RoadmapView { progress, bands, canvas }"
            in read("client/lib/features/project/roadmap_page.dart")
            and not re.search(r"lens", read("server/internal/api/roadmap.go"), re.I),
        ),
        (
            "2026-10-11-roadmap-dependency-edges.md",
            "roadmap dependency edges and the rebaseline flow",
            not re.search(r"depends_on", MIGRATIONS + "\n" + STORE, re.I),
        ),
        (
            "2026-10-11-focus-surface.md",
            "the Focus surface (per-session focus pointer, team Now)",
            "FocusNodeID" in read("server/internal/agentcontext/agentcontext.go")
            and not re.search(r"focus", ROUTES, re.I),
        ),
        (
            "2026-10-11-prd-autosave.md",
            "working-version autosave",
            not re.search(r"autosave|debounce", CLIENT, re.I),
        ),
    ]


def main():
    failures = []
    live = [(name, what) for name, what, holds in slice_premises() if holds]
    if not live:
        print("every parked slice is implemented; nothing to track")
        return 0
    issues = ROOT / "docs/issues"
    readme = read("docs/issues/README.md")
    for name, what in live + [(ENUMERATION, "the promise-gap enumeration")]:
        if not (issues / name).is_file():
            failures.append(f"{what}: no dated record docs/issues/{name}")
        elif f"]({name})" not in readme:
            failures.append(f"{what}: docs/issues/README.md does not list {name}")
    if f"](docs/issues/{ENUMERATION})" not in read("PRODUCT.md"):
        failures.append(
            "PRODUCT.md's status section does not link "
            f"docs/issues/{ENUMERATION}"
        )
    for f in failures:
        print(f"FAIL {f}")
    print(f"{len(live)} parked slices with live premises, "
          f"{len(failures)} tracking failures")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
