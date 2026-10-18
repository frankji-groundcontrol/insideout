"""Per-key suite for database-app-dsn — shape and placement, values never
printed (the DSN embeds a password; only its structure is asserted)."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestDatabaseAppDsn(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("database-app-dsn")
        if not cases:
            self.skipTest("no host on this machine carries database-app-dsn")
        for host, path, var in cases:
            self.assertEqual(var, "DATABASE_URL", f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertTrue(raw.startswith((b"postgres://", b"postgresql://")),
                            f"{host}: not a Postgres DSN")
            userinfo = raw.split(b"://", 1)[1].split(b"@", 1)[0]
            user = userinfo.split(b":", 1)[0]
            self.assertTrue(user.startswith(b"insideout_app"),
                            f"{host}: DSN must authenticate as the app role")
            self.assertTrue(
                user[len(b"insideout_app"):len(b"insideout_app") + 1]
                in (b":", b"."),
                f"{host}: role must end or take a pooler suffix here",
            )


if __name__ == "__main__":
    unittest.main()
