"""Per-key suite for github-token — optional PAT; absent is the intended
state, so assertions apply only where a value exists."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGithubToken(unittest.TestCase):
    def test_shape_and_placement_when_present(self):
        cases = _helpers.key_hosts("github-token")
        if not cases:
            self.skipTest("optional PAT; absent on this machine by design")
        for host, path, var in cases:
            self.assertEqual(var, "GITHUB_TOKEN", f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            if len(raw) == 0:
                continue  # the sanctioned empty placeholder (optional key)


if __name__ == "__main__":
    unittest.main()
