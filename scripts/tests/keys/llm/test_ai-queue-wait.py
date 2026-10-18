"""Per-key suite for ai-queue-wait — coach queue admission wait knob."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402

DURATION = r"^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)$"


class TestAiQueueWait(unittest.TestCase):
    def test_never_absent_and_shape(self):
        cases = _helpers.never_absent_cases("ai-queue-wait")
        if not cases:
            self.skipTest("no key store present on this machine")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_AI_QUEUE_WAIT",
                             f"{host}: wrong env var mapped")
            self.assertTrue(os.path.exists(path),
                            f"{host}: ai-queue-wait.key missing (run keys.py init)")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw).strip()
            if raw:
                _helpers.assert_regex(self, DURATION, raw.decode(),
                                      f"{host}: Go duration string (default 15s)")


if __name__ == "__main__":
    unittest.main()
