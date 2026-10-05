"""Per-key suite for client-port — compose host port for the web client."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


def _port_ok(raw: bytes) -> bool:
    return raw.isdigit() and 1 <= int(raw) <= 65535


class TestClientPort(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("client-port")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "CLIENT_PORT", f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: client-port.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path).strip()
            if raw:
                self.assertTrue(_port_ok(raw), f"{host}: TCP port (default 5173)")


if __name__ == "__main__":
    unittest.main()
