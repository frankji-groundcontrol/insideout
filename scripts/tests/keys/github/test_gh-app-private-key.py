"""Per-key suite for gh-app-private-key — the App PEM. The one multi-line
key: shape assertions only (PEM framing), never content comparison."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGhAppPrivateKey(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("gh-app-private-key")
        if not cases:
            self.skipTest("no host on this machine carries gh-app-private-key")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_GH_PRIVATE_KEY",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if _helpers.is_optional_blank(raw):
                continue  # blank = GitHub App integration disabled
            self.assertTrue(raw.startswith(b"-----BEGIN"),
                            f"{host}: PEM must open with a BEGIN line")
            self.assertTrue(b"PRIVATE KEY-----" in raw,
                            f"{host}: PEM header malformed")
            self.assertTrue(raw.rstrip().endswith(b"-----END PRIVATE KEY-----")
                            or raw.rstrip().endswith(b"-----END RSA PRIVATE KEY-----"),
                            f"{host}: PEM must close with an END line")
            self.assertTrue(b"\n" in raw,
                            f"{host}: the PEM is multi-line by design")
            self.assertGreater(len(raw), 800, f"{host}: implausibly small PEM")


if __name__ == "__main__":
    unittest.main()
