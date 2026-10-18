"""Per-key suite for postgres-owner-password — operator-filled by design,
so the assertions apply only where a value exists on this machine."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestPostgresOwnerPassword(unittest.TestCase):
    def test_shape_and_placement_when_present(self):
        cases = _helpers.key_hosts("postgres-owner-password")
        if not cases:
            self.skipTest("operator-filled; absent on this machine by design")
        for host, path, var in cases:
            self.assertEqual(var, "POSTGRES_OWNER_PASSWORD",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if len(raw) == 0:
                continue  # the sanctioned empty placeholder (operator-fills)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")


if __name__ == "__main__":
    unittest.main()
