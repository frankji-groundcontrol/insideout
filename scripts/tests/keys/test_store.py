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
        """(value path, fields) for every host x every shared manifest key."""
        for host in keys.discover_hosts(REPO):
            for y in keys.manifest_paths(REPO):
                module = os.path.basename(os.path.dirname(y))
                for name, fields in keys._parse_secrets_block(y).items():
                    path = os.path.join(REPO, "hosts", host, "keys", module,
                                        ".secrets", f"{name}.key")
                    yield path, fields

    def test_manifests_live_in_one_shared_set(self):
        # rv5-STRC-04: one manifest set under hosts/_manifests/keys/ serves
        # every host; a host folder keeps only host.yml, README and .secrets/.
        import glob
        shared = glob.glob(os.path.join(REPO, "hosts", "_manifests", "keys", "*", "*.yaml"))
        self.assertGreater(len(shared), 0, "no shared manifests under hosts/_manifests/keys/")
        per_host = sorted(
            os.path.relpath(p, REPO)
            for p in glob.glob(os.path.join(REPO, "hosts", "*", "keys", "*", "*.yaml"))
            if os.sep + "_manifests" + os.sep not in p)
        self.assertEqual(per_host, [], "per-host manifest copies remain")

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
        if checked == 0:
            self.skipTest("no .key on this machine (CI checkout)")

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
        if hosts_checked == 0:
            self.skipTest("no host present here (CI checkout)")

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

    # rv5-STRC-04: the manifests are a second catalogue of the variables in
    # .env.example. The drift recorded in
    # docs/issues/2026-11-21-key-manifests-one-set.md is settled; list a
    # name here only with a reason, and any unlisted difference fails.
    SKELETON_ONLY: dict = {}
    MANIFEST_ONLY: dict = {}

    def test_manifest_variables_match_the_skeleton(self):
        import env_catalog
        from pathlib import Path
        skeleton = {n for n, _, _, _ in env_catalog.parse_skeleton(Path(REPO) / ".env.example")}
        hosts = keys.discover_hosts(REPO)
        self.assertGreater(len(hosts), 0, "no hosts found")
        want = (skeleton - set(self.SKELETON_ONLY)) | set(self.MANIFEST_ONLY)
        for host in hosts:
            mapped = set(keys.key_map(REPO, host))
            self.assertEqual(sorted(mapped - want), [],
                             f"{host}: manifests map variables .env.example does not declare")
            self.assertEqual(sorted(want - mapped), [],
                             f"{host}: .env.example declares variables no manifest maps")


if __name__ == "__main__":
    unittest.main()
