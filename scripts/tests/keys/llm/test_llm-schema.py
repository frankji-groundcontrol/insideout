"""Per-key suite for llm-schema — the wire format selector."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestLlmSchema(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("llm-schema")
        if not cases:
            self.skipTest("no host on this machine overrides llm-schema")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_LLM_SCHEMA",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw).strip()
            if not raw.strip():
                continue  # blank = catalog default schema in effect
            self.assertTrue(raw in (b"messages", b"responses"),
                            f"{host}: schema is exactly messages|responses")


if __name__ == "__main__":
    unittest.main()
