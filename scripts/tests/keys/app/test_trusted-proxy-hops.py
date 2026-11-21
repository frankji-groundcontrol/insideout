"""Per-key suite for trusted-proxy-hops — appending reverse proxies in front of the server."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestTrustedProxyHops(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("trusted-proxy-hops")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_TRUSTED_PROXY_HOPS",
                             f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: trusted-proxy-hops.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw).strip()
            if raw:
                self.assertTrue(raw.isdigit(),
                                f"{host}: non-negative integer (server default 0)")


if __name__ == "__main__":
    unittest.main()
