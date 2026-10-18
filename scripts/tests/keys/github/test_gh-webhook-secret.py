"""Per-key suite for gh-webhook-secret — self-minted HMAC secret."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGhWebhookSecret(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("gh-webhook-secret")
        if not cases:
            self.skipTest("no host on this machine carries gh-webhook-secret")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_GH_WEBHOOK_SECRET",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if _helpers.is_optional_blank(raw):
                continue  # blank = signed GitHub webhook receiver disabled
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertGreaterEqual(len(raw), 32,
                                    f"{host}: below openssl-rand-hex-32 strength")


if __name__ == "__main__":
    unittest.main()
