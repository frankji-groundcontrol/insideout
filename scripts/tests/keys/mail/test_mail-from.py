"""Per-key suite for mail-from — the display-name From sender."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestMailFrom(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("mail-from")
        if not cases:
            self.skipTest("no host on this machine overrides mail-from")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_MAIL_FROM",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertTrue(b"@" in raw, f"{host}: a sender carries an address")


if __name__ == "__main__":
    unittest.main()
