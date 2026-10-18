#!/usr/bin/env python3
"""Pin the privacy scrub of review-4 wave 0 (rv4-PUB-01, PM-11, PUB-05, PUB-10, PUB-12).

Usage: python3 -I scripts/tests/check_redaction_addresses.py

Stdlib only. The address rule of scripts/check-redaction.sh is driven against
a throwaway git repository, so the test needs no real address: a synthetic
address on a domain nobody reserved must fail the guard without being echoed
back, and every documented placeholder form must pass. Addresses are built
around AT so this file never spells one itself and the guard stays green on
its own tree.
"""
import os
import pathlib
import subprocess
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
GUARD = ROOT / "scripts" / "check-redaction.sh"
AT = "@"

# The guard reads the hosted host from this constant and exits 2 without it.
DEFAULT_BASE = "server/internal/apiclient/default_base.go"
DEFAULT_BASE_TEXT = 'package apiclient\n\nconst DefaultBase = "https://hosted.example.test/api/v1"\n'


def addr(local, domain):
    return f"{local}{AT}{domain}"


def env():
    # A hook or CI wrapper may export GIT_DIR and friends; the throwaway repo
    # must not inherit them.
    return {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}


def run(cmd, cwd):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=env())


def guard_on(files):
    """Run the guard in a throwaway repo whose tracked files are `files`."""
    files = {DEFAULT_BASE: DEFAULT_BASE_TEXT, **files}
    with tempfile.TemporaryDirectory() as tmp:
        for name, text in files.items():
            path = pathlib.Path(tmp, name)
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8")
        run(["git", "init", "-q"], tmp)
        run(["git", "add", "--", *files], tmp)
        return run(["sh", str(GUARD)], tmp)


class AddressRule(unittest.TestCase):
    """rv4-PUB-01: the guard knew one literal address and was blind to the rest."""

    def test_unreserved_address_fails_in_any_tracked_file_and_is_not_echoed(self):
        local, domain = "zq-probe", "acme-corp.io"
        for path in ("docs/note.md", "server/x.go", "client/lib/x.dart", "config/app.yaml"):
            with self.subTest(path=path):
                r = guard_on({path: f"// mail {addr(local, domain)} please\n"})
                self.assertEqual(r.returncode, 1, r.stderr)
                self.assertIn(f"{path}:1", r.stderr)
                for value in (local, domain):
                    self.assertNotIn(value, r.stderr, "the guard must not print the value")

    def test_match_ignores_case(self):
        r = guard_on({"docs/n.md": addr("Zq-Probe", "ACME-Corp.IO") + "\n"})
        self.assertEqual(r.returncode, 1, r.stderr)

    def test_placeholder_forms_pass(self):
        allowed = [
            addr("noreply", "mail.acme-corp.io"),  # the local part says nobody reads it
            addr("zq-probe", "example.com"),
            addr("zq-probe", "mail.example.org"),
            addr("zq-probe", "db.example"),
            addr("zq-probe", "mail.test"),
            addr("zq-probe", "svc.invalid"),
            addr("zq-probe", "insideout.local"),
            addr("zq-probe", "box.localhost"),
            addr("onboarding", "resend.dev"),  # the mail provider's sandbox
            addr("a", "x.com"),  # fixture domains of the Go auth tests
            addr("A", "X.COM"),
            addr("a", "x.co"),
            addr("a", "b.co"),
            "icon" + AT + "2x.png",  # retina asset names are not addresses
            "logo" + AT + "3x.jpg",
        ]
        for text in allowed:
            with self.subTest(text=text):
                r = guard_on({"docs/n.md": f"see {text} here\n"})
                self.assertEqual((r.returncode, r.stderr), (0, ""))

    def test_lookalikes_of_the_placeholder_forms_fail(self):
        lookalikes = [
            addr("xnoreply", "acme-corp.io"),  # noreply must be the whole local part
            addr("zq-probe", "notexample.com"),  # example must be a whole label
            addr("zq-probe", "example.com.acme-corp.io"),
            addr("zq-probe", "xx.com"),  # fixture domains match exactly
            addr("zq-probe", "mail.x.com"),
            addr("zq-probe", "notresend.dev"),
            addr("zq-probe", "resend.dev.acme-corp.io"),
        ]
        for text in lookalikes:
            with self.subTest(text=text):
                r = guard_on({"docs/n.md": f"see {text} here\n"})
                self.assertEqual(r.returncode, 1, r.stderr)


class TrackedTree(unittest.TestCase):
    """Facts about this checkout that the scrub established and must not drift back."""

    def test_guard_passes_on_this_checkout(self):
        # rv4-PM-11: the owner-domain account address sat in a dated record.
        r = run(["sh", str(GUARD)], ROOT)
        self.assertEqual(r.returncode, 0, r.stderr)


if __name__ == "__main__":
    unittest.main()
