#!/usr/bin/env python3
"""Fail when the Go toolchain floor drifts: go.mod vs image vs installer prose.

Usage: python3 -I scripts/tests/check_go_floor.py

The floor is the `go` directive in server/go.mod. server/Dockerfile must
build on a golang base at least that high (the golang images run
GOTOOLCHAIN=local, so an older base cannot compile the module and the
image build dies at `go mod download` — BUG-005, review-3 ARCH-01), and
the prose files a fresh installer reads first must not promise an older
minimum (review-3 ARCH-02). A file that states no number (it points at
go.mod instead) passes. Two more pins are held to the floor (review-4
PUB-09): the GO_VERSION the install skill downloads, and the Ubuntu CI
verify step, which must ask the go command about server/go.mod instead of
grepping `go version` for a hand-written range of releases (a range goes
stale the day the floor moves). Exits 1 naming every offending pin.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]

# The five files a from-source installer reads first. None may state a Go
# minimum older than the go.mod floor; stating none at all is fine.
PROSE_FILES = [
    "README.md",
    "docs/usage/installation.md",
    "docs/architecture/backend.md",
    "scripts/install.sh",
    "skills/insideout-setup/references/modules.md",
]

# How those files phrase a Go minimum. Each capture is a stated floor.
MINIMUM_PATTERNS = [
    r"Go (\d+\.\d+)\+",  # "needs Go 1.26+", "## 3. Go 1.26+"
    r"minimum (\d+\.\d+)",  # "**Go**: minimum 1.26"
    r"go(\d+\.\d+)` or newer",  # "must print `go1.26` or newer"
]


def go_mod_floor():
    text = (ROOT / "server/go.mod").read_text(encoding="utf-8")
    match = re.search(r"^go (\d+\.\d+(?:\.\d+)?)", text, re.M)
    if not match:
        print("no `go` directive found in server/go.mod", file=sys.stderr)
        sys.exit(2)
    return match.group(1)


def image_go_version():
    text = (ROOT / "server/Dockerfile").read_text(encoding="utf-8")
    match = re.search(r"^FROM golang:([\d.]+)", text, re.M)
    if not match:
        print("no golang base image found in server/Dockerfile", file=sys.stderr)
        sys.exit(2)
    return match.group(1).rstrip(".")


def version_tuple(version):
    # "1.26" and "1.26.0" are the same floor: pad before comparing.
    parts = [int(p) for p in version.split(".")]
    return tuple((parts + [0, 0])[:3])


def stale_prose_minimums(floor):
    """Every (file, line, stated minimum) in the installer prose below the floor."""
    stale = []
    for name in PROSE_FILES:
        for no, line in enumerate(
            (ROOT / name).read_text(encoding="utf-8").splitlines(), start=1
        ):
            for pattern in MINIMUM_PATTERNS:
                for match in re.finditer(pattern, line):
                    stated = match.group(1)
                    if version_tuple(stated) < version_tuple(floor):
                        stale.append((name, no, stated))
    return stale


def installer_go_version():
    """The Go release the install skill's Ubuntu block downloads (GO_VERSION=x.y.z)."""
    text = (ROOT / "skills/insideout-setup/references/modules.md").read_text(encoding="utf-8")
    match = re.search(r"^GO_VERSION=(\d+\.\d+\.\d+)", text, re.M)
    if not match:
        print("no GO_VERSION=x.y.z line in skills/insideout-setup/references/modules.md", file=sys.stderr)
        sys.exit(2)
    return match.group(1)


def ci_hardcoded_ranges():
    """Line numbers in ubuntu-install.yml that hard-code a range of Go releases (go1\\.(2[6-9]|...))."""
    text = (ROOT / ".github/workflows/ubuntu-install.yml").read_text(encoding="utf-8")
    return [no for no, line in enumerate(text.splitlines(), start=1) if "go1\\.(" in line]


if __name__ == "__main__":
    floor = go_mod_floor()
    image = image_go_version()
    failures = 0
    if version_tuple(image) < version_tuple(floor):
        print(
            f"server/Dockerfile builds on golang:{image} but server/go.mod "
            f"requires go >= {floor} — the image build fails at `go mod "
            "download`; bump the base image",
            file=sys.stderr,
        )
        failures += 1
    for name, no, stated in stale_prose_minimums(floor):
        print(
            f"{name}:{no}: states Go minimum {stated} but server/go.mod "
            f"requires >= {floor}",
            file=sys.stderr,
        )
        failures += 1
    pinned = installer_go_version()
    if version_tuple(pinned) < version_tuple(floor):
        print(
            f"skills/insideout-setup/references/modules.md installs Go {pinned} but "
            f"server/go.mod requires >= {floor}",
            file=sys.stderr,
        )
        failures += 1
    for no in ci_hardcoded_ranges():
        print(
            f".github/workflows/ubuntu-install.yml:{no}: greps `go version` for a "
            "hand-written range of Go releases; derive the floor from server/go.mod "
            "(GOTOOLCHAIN=local go list -m run in server/)",
            file=sys.stderr,
        )
        failures += 1
    if failures:
        sys.exit(1)
    print(
        f"go floor ok: golang:{image} >= go.mod's go {floor}, prose in sync, "
        f"installer pins go{pinned}, CI derives the floor"
    )
