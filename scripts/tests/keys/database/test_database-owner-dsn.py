"""Per-key suite for database-owner-dsn — shape and placement, values never
printed."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestDatabaseOwnerDsn(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("database-owner-dsn")
        if not cases:
            self.skipTest("no host on this machine carries database-owner-dsn")
        for host, path, var in cases:
            self.assertEqual(var, "DATABASE_OWNER_URL",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            self.assertTrue(raw.startswith((b"postgres://", b"postgresql://")),
                            f"{host}: not a Postgres DSN")
            userinfo = raw.split(b"://", 1)[1].split(b"@", 1)[0]
            user = userinfo.split(b":", 1)[0]
            self.assertTrue(user.startswith(b"insideout_owner"),
                            f"{host}: DSN must authenticate as the owner role")
            self.assertIn(user[len(b"insideout_owner"):len(b"insideout_owner") + 1],
                          (b":", b"."),
                          f"{host}: role must end or take a pooler suffix here")
            self.assertNotIn(b"\n", raw, f"{host}: must be one line")


if __name__ == "__main__":
    unittest.main()
