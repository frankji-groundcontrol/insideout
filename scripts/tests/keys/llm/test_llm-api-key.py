"""Per-key suite for llm-api-key — the PRD coach's model key."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestLlmApiKey(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("llm-api-key")
        if not cases:
            self.skipTest("no host on this machine carries llm-api-key")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_LLM_API_KEY",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            self.assertGreaterEqual(len(raw), 20, f"{host}: implausibly short")
            self.assertNotIn(b"\n", raw, f"{host}: must be one line")


if __name__ == "__main__":
    unittest.main()
