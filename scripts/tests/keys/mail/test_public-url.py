"""Per-key suite for public-url — the public origin of emailed links."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestPublicUrl(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("public-url")
        if not cases:
            self.skipTest("no host on this machine overrides public-url")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_PUBLIC_URL",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            self.assertTrue(raw.startswith((b"https://", b"http://")),
                            f"{host}: must be an absolute http(s) origin")
            self.assertNotIn(b"\n", raw, f"{host}: must be one line")


if __name__ == "__main__":
    unittest.main()
