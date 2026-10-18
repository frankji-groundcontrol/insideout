"""Per-key suite for llm-model — the model id the coach requests."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestLlmModel(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("llm-model")
        if not cases:
            self.skipTest("no host on this machine overrides llm-model")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_LLM_MODEL",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            if not raw.strip():
                continue  # blank = catalog default model in effect
            self.assertGreater(len(raw), 0, f"{host}: model id must not be empty")
            self.assertTrue(b" " not in raw, f"{host}: model ids carry no spaces")


if __name__ == "__main__":
    unittest.main()
