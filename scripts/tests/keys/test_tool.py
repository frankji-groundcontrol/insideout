"""Unit tests for scripts/keys.py — the module-grouped key store audit/fill
bridge (tool behaviors: manifest parsing, no-value-leak, empty-only fills,
multi-host scoping).

Fixtures build miniature hosts/ trees in a temp dir using the same shape
the real tree uses (hosts/<host>/keys/<module>/ holding one or more YAML
manifests plus a shared .secrets/ dir). Organized under scripts/tests/keys/
mirroring the store; the per-key suites live in the module folders beside
this file; store-wide invariants live in test_store.py.
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

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(os.path.dirname(HERE))
for p in (HERE, SCRIPTS):
    if p not in sys.path:
        sys.path.insert(0, p)

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
        self.assertTrue(a.read_text() == "value-of-a-dsn",
                        "filled keys must never be overwritten")
        self.assertTrue(b.read_text() == "fresh-value")
        self.assertEqual(stat.S_IMODE(b.stat().st_mode), 0o600)
        self.assertNotIn("fresh-value", out.getvalue(), "fill must never print a value")

    def test_fill_skipsPlaceholdersAndMissing(self):
        self.build_default()
        env = self.root / "env"
        env.write_text("A_DSN=keep\nB_KEY=changeme-please\n")
        keys.fill(str(self.root), "testhost", str(env), dry_run=False)
        b = self.root / "hosts/testhost/keys/db/.secrets/b-key.key"
        self.assertTrue(b.read_text() == "", "placeholders must not be written")

    def test_audit_flagsUnmappedSecretShapedEnvNames(self):
        self.build_default()
        env = self.root / "env"
        env.write_text("A_DSN=x\nB_KEY=y\nMY_STALE_TOKEN=z\nPLAIN_CONFIG=1\n")
        out = io.StringIO()
        with redirect_stdout(out):
            keys.audit(str(self.root), "testhost", str(env))
        self.assertIn("MY_STALE_TOKEN", out.getvalue())
        self.assertNotIn("PLAIN_CONFIG", out.getvalue())

    def test_env_state_displayNameSender_is_not_a_placeholder(self):
        # A Resend-style From value carries display-name brackets
        # (Name <addr@host>); only bracket tokens WITHOUT an @ are
        # skeleton placeholders (<password>, <your-domain>).
        self.assertEqual(keys.env_state("InsideOut <noreply@example.com>"), "set")
        self.assertEqual(keys.env_state("postgres://u:<password>@h/db"), "placeholder")

    def test_init_creates_empty_placeholders_without_touching_filled(self):
        self.build_default()
        env = dict(os.environ, INSIDEOUT_KEYS_HOST="testhost")
        r = subprocess.run(
            [sys.executable, os.path.join(SCRIPTS, "keys.py"),
             "init", "--root", str(self.root)],
            capture_output=True, text=True, env=env,
        )
        self.assertEqual(r.returncode, 0, r.stderr)
        a = self.root / "hosts/testhost/keys/db/.secrets/a-dsn.key"
        b = self.root / "hosts/testhost/keys/db/.secrets/b-key.key"
        self.assertTrue(a.read_text() == "value-of-a-dsn",
                        "init must never touch a filled key")
        self.assertTrue(b.exists(), "the never-absent rule: every mapped kind gets a file")
        self.assertTrue(b.read_text() == "",
                        "placeholder starts empty (fallback in effect)")
        self.assertEqual(stat.S_IMODE(b.stat().st_mode), 0o600)
        # idempotent
        r2 = subprocess.run(
            [sys.executable, os.path.join(SCRIPTS, "keys.py"),
             "init", "--root", str(self.root)],
            capture_output=True, text=True, env=env,
        )
        self.assertEqual(r2.returncode, 0, r2.stderr)
        self.assertTrue(a.read_text() == "value-of-a-dsn")

    def test_cli_audit_end_to_end(self):
        self.build_default()
        r = subprocess.run(
            [sys.executable, os.path.join(SCRIPTS, "keys.py"),
             "audit", "--root", str(self.root)],
            capture_output=True, text=True,
        )
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("2 key kinds audited", r.stdout)


def make_host(root: Path, host: str, module: str, setting: str,
              name: str, env: str) -> None:
    """A one-key host tree: manifest + a filled .key, for multi-host tests."""
    mod = root / "hosts" / host / "keys" / module
    (mod / ".secrets").mkdir(parents=True, mode=0o700)
    (root / "hosts" / host / "host.yml").write_text(f"name: {host}\n")
    (mod / f"{setting}.yaml").write_text(
        FLOW_MANIFEST.format(module=module, setting=setting,
                             b_name=name, b_env=env))
    k = mod / ".secrets" / f"{name}.key"
    fd = os.open(k, os.O_WRONLY | os.O_CREAT, 0o600)
    with os.fdopen(fd, "w") as fh:
        fh.write(f"value-of-{name}")
    os.chmod(k, 0o600)


class TestMultiHostScoping(unittest.TestCase):
    """With more than one host, fill must be scoped and refuse to guess."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.addCleanup(self.tmp.cleanup)
        self.keys_py = os.path.join(SCRIPTS, "keys.py")
        make_host(self.root, "hosta", "db", "main", "a-dsn", "A_DSN")
        make_host(self.root, "hostb", "db", "main", "b-dsn", "B_DSN")
        (self.root / "env").write_text("A_DSN=new-a\nB_DSN=new-b\n")
        # Both keys start filled; a scoped fill must keep them (empty-only)
        # while proving the *other* host's tree is never even a candidate.

    def test_cli_refuses_without_scoping(self):
        # envfile before --root: Python 3.9's argparse rejects an
        # optional interleaved between two positionals.
        r = subprocess.run(
            [sys.executable, self.keys_py, "fill",
             str(self.root / "env"), "--root", str(self.root)],
            capture_output=True, text=True,
        )
        self.assertNotEqual(r.returncode, 0)
        self.assertIn("INSIDEOUT_KEYS_HOST", r.stderr)

    def test_scoped_fill_touches_only_its_host(self):
        env = dict(os.environ, INSIDEOUT_KEYS_HOST="hosta")
        r = subprocess.run(
            [sys.executable, self.keys_py, "fill",
             str(self.root / "env"), "--root", str(self.root)],
            capture_output=True, text=True, env=env,
        )
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("hosts/hosta/keys", r.stdout)
        self.assertNotIn("hosts/hostb/keys", r.stdout,
                         "a scoped fill must never name another host's paths")
        a = (self.root / "hosts/hosta/keys/db/.secrets/a-dsn.key").read_text()
        b = (self.root / "hosts/hostb/keys/db/.secrets/b-dsn.key").read_text()
        self.assertTrue(a == "value-of-a-dsn", "filled keys are kept, not rewritten")
        self.assertTrue(b == "value-of-b-dsn",
                        "the other host's values are untouched")


if __name__ == "__main__":
    unittest.main()
