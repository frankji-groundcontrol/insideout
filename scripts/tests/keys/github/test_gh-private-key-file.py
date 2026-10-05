"""Per-key suite for gh-private-key-file — the PEM store pointer
(kind: path). The strong assertion: the pointed-at file must exist."""

import os
import stat
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestGhPrivateKeyFile(unittest.TestCase):
    def test_pointer_resolves_to_the_store_pem(self):
        cases = _helpers.key_hosts("gh-private-key-file")
        if not cases:
            self.skipTest("no host on this machine uses the _FILE pointer")
        for host, path, var in cases:
            self.assertEqual(var, "INSIDEOUT_GH_PRIVATE_KEY_FILE",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            target = _helpers.read_bytes(path).decode().strip()
            self.assertTrue(os.path.exists(target),
                            f"{host}: pointer target does not exist")
            self.assertTrue(target.endswith("gh-app-private-key.key"),
                            f"{host}: pointer must name the store PEM")
            self.assertIn(os.sep + "keys" + os.sep, target,
                          f"{host}: pointer should stay inside the key store")


if __name__ == "__main__":
    unittest.main()
