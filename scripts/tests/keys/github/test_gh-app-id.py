"""Per-key suite for gh-app-id — the GitHub App numeric id."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGhAppId(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("gh-app-id")
        if not cases:
            self.skipTest("no host on this machine carries gh-app-id")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_GH_APP_ID",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            self.assertTrue(raw.strip().isdigit(),
                            f"{host}: the App id is numeric")
            self.assertGreater(len(raw.strip()), 0)


if __name__ == "__main__":
    unittest.main()
