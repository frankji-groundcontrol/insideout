"""Unit tests for scripts/keys.py — the module-grouped key store audit/fill bridge.

Fixtures build a miniature hosts/ tree in a temp dir using the same shape
the real tree uses (hosts/<host>/keys/<module>/ holding one or more YAML
manifests plus a shared .secrets/ dir); assertions pin the manifest
parsing (block and flow entry styles, multiple yamls per module), the
no-value-leak guarantee, empty-only fills, 0600 modes, and the
unmapped-secret report.
"""

import io
import os
import stat
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import keys  # noqa: E402

BLOCK_MANIFEST = """# Metadata only — values live in .secrets/<name>.key, never here.
schema_version: 1
host: testhost
module: {module}
setting: {setting}
secrets:
  {a_name}:
    var: {a_env}
    kind: value
"""

FLOW_MANIFEST = """schema_version: 1
host: testhost
module: {module}
setting: {setting}
secrets:
  {b_name}: {{ var: {b_env}, kind: value }}
"""


def make_tree(root: Path, module: str, setting: str,
              a_name: str, a_env: str, a_filled: bool,
              b_name: str, b_env: str, b_filled: bool) -> None:
    mod = root / "hosts" / "testhost" / "keys" / module
    (mod / ".secrets").mkdir(parents=True, mode=0o700)
    (root / "hosts" / "testhost" / "host.yml").write_text("name: testhost\n")
    (mod / f"{setting}-block.yaml").write_text(
        BLOCK_MANIFEST.format(module=module, setting=setting,
                              a_name=a_name, a_env=a_env))
    (mod / f"{setting}-flow.yaml").write_text(
        FLOW_MANIFEST.format(module=module, setting=setting,
                             b_name=b_name, b_env=b_env))
    for name, filled in [(a_name, a_filled), (b_name, b_filled)]:
        k = mod / ".secrets" / f"{name}.key"
        fd = os.open(k, os.O_WRONLY | os.O_CREAT, 0o600)
        with os.fdopen(fd, "w") as fh:
            if filled:
                fh.write(f"value-of-{name}")
        os.chmod(k, 0o600)


class TestKeys(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.addCleanup(self.tmp.cleanup)

    def build_default(self):
        make_tree(self.root, module="db", setting="main",
                  a_name="a-dsn", a_env="A_DSN", a_filled=True,
                  b_name="b-key", b_env="B_KEY", b_filled=False)

    def test_key_map_parsesBlockAndFlowAcrossManifests(self):
        self.build_default()
        m = keys.key_map(str(self.root), "testhost")
        self.assertEqual(m["A_DSN"][0], "db/a-dsn")
        self.assertEqual(m["B_KEY"][0], "db/b-key")
        self.assertTrue(m["A_DSN"][1].endswith("keys/db/.secrets/a-dsn.key"))
        self.assertTrue(m["B_KEY"][1].endswith("keys/db/.secrets/b-key.key"))

    def test_multipleModules_shareTheKeysRoot(self):
        self.build_default()
        make_tree(self.root, module="mail", setting="resend",
                  a_name="c-token", a_env="C_TOKEN", a_filled=False,
                  b_name="d-secret", b_env="D_SECRET", b_filled=False)
        m = keys.key_map(str(self.root), "testhost")
        self.assertEqual(
            sorted(v[0].split("/")[0] for v in m.values()),
            ["db", "db", "mail", "mail"],
        )

    def test_audit_reportsFilledAndEmpty_withoutValues(self):
        self.build_default()
        out = io.StringIO()
        with redirect_stdout(out):
            keys.audit(str(self.root), "testhost", None)
        text = out.getvalue()
        self.assertIn("filled", text)
        self.assertIn("empty", text)
        self.assertNotIn("value-of-a-dsn", text, "audit must never print a value")

    def test_fill_writesOnlyEmptyKeys_atMode0600(self):
        self.build_default()
        env = self.root / "env"
        env.write_text("A_DSN=overwritten\nB_KEY=fresh-value\n# comment\n")
        out = io.StringIO()
        with redirect_stdout(out):
            keys.fill(str(self.root), "testhost", str(env), dry_run=False)
        a = self.root / "hosts/testhost/keys/db/.secrets/a-dsn.key"
        b = self.root / "hosts/testhost/keys/db/.secrets/b-key.key"
        self.assertEqual(a.read_text(), "value-of-a-dsn", "filled keys must never be overwritten")
        self.assertEqual(b.read_text(), "fresh-value")
        self.assertEqual(stat.S_IMODE(b.stat().st_mode), 0o600)
        self.assertNotIn("fresh-value", out.getvalue(), "fill must never print a value")

    def test_fill_skipsPlaceholdersAndMissing(self):
        self.build_default()
        env = self.root / "env"
        env.write_text("A_DSN=keep\nB_KEY=changeme-please\n")
        keys.fill(str(self.root), "testhost", str(env), dry_run=False)
        b = self.root / "hosts/testhost/keys/db/.secrets/b-key.key"
        self.assertEqual(b.read_text(), "", "placeholders must not be written")

    def test_audit_flagsUnmappedSecretShapedEnvNames(self):
        self.build_default()
        env = self.root / "env"
        env.write_text("A_DSN=x\nB_KEY=y\nMY_STALE_TOKEN=z\nPLAIN_CONFIG=1\n")
        out = io.StringIO()
        with redirect_stdout(out):
            keys.audit(str(self.root), "testhost", str(env))
        self.assertIn("MY_STALE_TOKEN", out.getvalue())
        self.assertNotIn("PLAIN_CONFIG", out.getvalue())

    def test_cli_audit_end_to_end(self):
        self.build_default()
        r = subprocess.run(
            [sys.executable, os.path.join(os.path.dirname(__file__), "keys.py"),
             "audit", "--root", str(self.root)],
            capture_output=True, text=True,
        )
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("2 key kinds audited", r.stdout)


if __name__ == "__main__":
    unittest.main()
