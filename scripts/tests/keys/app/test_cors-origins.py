"""Per-key suite for cors-origins — extra allowed CORS origins."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestCorsOrigins(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("cors-origins")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_CORS_ORIGINS",
                             f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: cors-origins.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw).strip()
            if raw:
                for origin in raw.decode().split(","):
                    _helpers.assert_regex(
                        self, r"^https?://[^/]+$", origin.strip(),
                        f"{host}: origin entries are scheme://host[:port]",
                    )


if __name__ == "__main__":
    unittest.main()
