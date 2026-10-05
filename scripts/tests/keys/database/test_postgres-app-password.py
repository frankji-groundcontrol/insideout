"""Per-key suite for postgres-app-password (bundled compose stack)."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestPostgresAppPassword(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("postgres-app-password")
        if not cases:
            self.skipTest("no host on this machine carries postgres-app-password")
        for host, path, var in cases:
            self.assertEqual(var, "POSTGRES_APP_PASSWORD",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            self.assertGreaterEqual(len(raw), 16, f"{host}: implausibly short")
            self.assertNotIn(b"\n", raw, f"{host}: must be one line")


if __name__ == "__main__":
    unittest.main()
