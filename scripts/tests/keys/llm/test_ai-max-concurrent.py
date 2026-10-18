"""Per-key suite for ai-max-concurrent — coach parallelism knob."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestAiMaxConcurrent(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("ai-max-concurrent")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_AI_MAX_CONCURRENT",
                             f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: ai-max-concurrent.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw).strip()
            if raw:
                self.assertTrue(raw.isdigit() and int(raw) > 0,
                                f"{host}: positive integer (default 4)")


if __name__ == "__main__":
    unittest.main()
