"""Cross-cutting invariants of the real hosts/ key store.

Applies to every host and every manifest key present on this machine
(CI-safe: file checks apply only where a `.secrets/` dir exists here).
"""

import os
import stat
import subprocess
import sys
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(os.path.dirname(HERE))
REPO = os.path.dirname(SCRIPTS)
for p in (HERE, SCRIPTS):
    if p not in sys.path:
        sys.path.insert(0, p)

import keys  # noqa: E402
import _helpers  # noqa: E402


class TestStoreInvariants(unittest.TestCase):
    def manifest_entries(self):
        import glob
        pattern = os.path.join(REPO, "hosts", "*", "keys", "*", "*.yaml")
        for y in sorted(glob.glob(pattern)):
            for name, fields in keys._parse_secrets_block(y).items():
                path = os.path.join(os.path.dirname(y), ".secrets", f"{name}.key")
                yield path, fields

    def test_every_manifest_key_honors_store_invariants(self):
        checked = 0
        for path, fields in self.manifest_entries():
            if not os.path.exists(path):
                continue  # host not present on this machine / CI checkout
            checked += 1
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600,
                             f"{path} must be 0600")
            permissive = "true" in (fields.get("operator_fills", ""),
                                    fields.get("optional", "")) \
                or "fallback" in fields  # present (even empty) = blank is legal
            if not permissive:
                self.assertGreater(os.path.getsize(path), 0,
                                   f"{path} is mapped but empty and not operator-filled")
            secrets_dir = os.path.dirname(path)
            self.assertEqual(stat.S_IMODE(os.stat(secrets_dir).st_mode), 0o700,
                             f"{secrets_dir} must be 0700")
        self.assertGreater(checked, 0, "expected at least one .key on this machine")

    def test_never_absent_rule(self):
        """A host present on this machine (>=1 .key exists) must have a
        `.key` file for EVERY mapped kind: a knob may be blank (its
        fallback is in effect) but never missing."""
        hosts_checked = 0
        by_host = {}
        for path, _fields in self.manifest_entries():
            host = path.split(os.sep + "keys" + os.sep)[0]
            by_host.setdefault(host, []).append(path)
        for host, paths in sorted(by_host.items()):
            if not any(os.path.exists(p) for p in paths):
                continue  # host not present on this machine / CI checkout
            hosts_checked += 1
            missing = [p for p in paths if not os.path.exists(p)]
            self.assertEqual(missing, [],
                             f"{host}: mapped kinds with no .key file (run keys.py init): "
                             f"{missing}")
        self.assertGreater(hosts_checked, 0, "expected at least one host present here")

    def test_git_ignores_every_secrets_path(self):
        for path, _fields in self.manifest_entries():
            if not os.path.exists(path):
                continue
            r = subprocess.run(["git", "-C", REPO, "check-ignore", "-q", path],
                               capture_output=True)
            self.assertEqual(r.returncode, 0, f"git does not ignore {path}")

    def test_host_yml_name_matches_folder(self):
        import glob
        seen = 0
        for p in sorted(glob.glob(os.path.join(REPO, "hosts", "*", "host.yml"))):
            seen += 1
            folder = os.path.basename(os.path.dirname(p))
            with open(p) as fh:
                content = fh.read()
            self.assertIn(f"name: {folder}", content,
                          f"{p} must name its own folder")
        self.assertGreater(seen, 0, "no hosts found")


if __name__ == "__main__":
    unittest.main()
