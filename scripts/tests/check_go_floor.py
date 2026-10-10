#!/usr/bin/env python3
"""Fail when the server image's Go toolchain is older than go.mod's floor.

Usage: python3 -I scripts/tests/check_go_floor.py

The floor is the `go` directive in server/go.mod. The golang base images
run GOTOOLCHAIN=local, so a base image older than the directive cannot
compile the module and the image build dies at `go mod download`
(BUG-005, review-3 ARCH-01). Exits 1 naming the offending pin.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]


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


if __name__ == "__main__":
    floor = go_mod_floor()
    image = image_go_version()
    if version_tuple(image) < version_tuple(floor):
        print(
            f"server/Dockerfile builds on golang:{image} but server/go.mod "
            f"requires go >= {floor} — the image build fails at `go mod "
            "download`; bump the base image",
            file=sys.stderr,
        )
        sys.exit(1)
    print(f"go floor ok: golang:{image} >= go.mod's go {floor}")
