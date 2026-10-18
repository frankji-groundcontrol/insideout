"""Per-key suite for gh-client-secret — kept credential, no reader yet."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGhClientSecret(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("gh-client-secret")
        if not cases:
            self.skipTest("no host on this machine carries gh-client-secret")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_GH_CLIENT_SECRET",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if _helpers.is_optional_blank(raw):
                continue  # blank = unused future OAuth consumer
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertGreaterEqual(len(raw), 20, f"{host}: implausibly short")


if __name__ == "__main__":
    unittest.main()
