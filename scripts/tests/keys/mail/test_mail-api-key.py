"""Per-key suite for mail-api-key — Resend key for auth email."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestMailApiKey(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("mail-api-key")
        if not cases:
            self.skipTest("no host on this machine carries mail-api-key")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_MAIL_API_KEY",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if _helpers.is_optional_blank(raw):
                continue  # blank = supported development-mail lane
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertTrue(raw.startswith(b"re_"),
                            f"{host}: Resend keys carry the re_ prefix")
            self.assertGreaterEqual(len(raw), 20, f"{host}: implausibly short")


if __name__ == "__main__":
    unittest.main()
