"""Per-key suite for access-ttl — access-token lifetime knob."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402

DURATION = r"^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)$"


class TestAccessTtl(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("access-ttl")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_ACCESS_TTL",
                             f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: access-ttl.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path).strip()
            if raw:
                self.assertRegex(raw.decode(), DURATION,
                                 f"{host}: Go duration string (default 15m)")


if __name__ == "__main__":
    unittest.main()
