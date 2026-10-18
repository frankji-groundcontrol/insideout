"""Per-key suite for database-owner-dsn — shape and placement, values never
printed; and what `env.sh check` says when the owner DSN is empty."""

import os
import stat
import subprocess
import tempfile
import unittest

import sys as _sys
_sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import _helpers  # noqa: E402


class TestDatabaseOwnerDsn(unittest.TestCase):
    def test_shape_and_placement(self):
        cases = _helpers.key_hosts("database-owner-dsn")
        if not cases:
            self.skipTest("no host on this machine carries database-owner-dsn")
        for host, path, var in cases:
            self.assertEqual(var, "DATABASE_OWNER_URL",
                             f"{host}: wrong env var mapped")
            self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)
            raw = _helpers.read_bytes(path)
            _helpers.assert_single_line(self, raw, f"{host}: must be one line")
            raw = _helpers.without_terminal_newline(raw)
            self.assertTrue(raw.startswith((b"postgres://", b"postgresql://")),
                            f"{host}: not a Postgres DSN")
            userinfo = raw.split(b"://", 1)[1].split(b"@", 1)[0]
            user = userinfo.split(b":", 1)[0]
            self.assertTrue(user.startswith(b"insideout_owner"),
                            f"{host}: DSN must authenticate as the owner role")
            self.assertTrue(
                user[len(b"insideout_owner"):len(b"insideout_owner") + 1]
                in (b":", b"."),
                f"{host}: role must end or take a pooler suffix here",
            )


ENV_SH = os.path.join(_helpers.SCRIPTS, "env.sh")
# A placeholder-free value for every REQUIRED variable, so only the DSN pair
# under test can move the verdict. Example strings, not secrets.
BASE_ENV = {
    "INSIDEOUT_JWT_SECRET": "0123456789abcdef0123456789abcdef0123456789abcdef",
    "POSTGRES_APP_PASSWORD": "example-app-password",
    "POSTGRES_OWNER_PASSWORD": "example-owner-password",
}


def run_check(env_vars):
    """`env.sh check` against a throwaway env file in a clean environment;
    returns (exit status, stdout)."""
    with tempfile.TemporaryDirectory() as d:
        path = os.path.join(d, "env")
        with open(path, "w") as fh:
            for key, value in {**BASE_ENV, **env_vars}.items():
                fh.write(f"{key}={value}\n")
        proc = subprocess.run(
            ["bash", ENV_SH, "check"], capture_output=True, text=True, timeout=60,
            env={"PATH": os.environ.get("PATH", ""), "HOME": d,
                 "INSIDEOUT_ENV_FILE": path},
        )
    return proc.returncode, proc.stdout


class TestEnvCheckOwnerDsn(unittest.TestCase):
    """rv4-FULL-06: the bundled Postgres' DATABASE_URL is insideout_app, which
    cannot apply migrations, so compose with only the app DSN dies at migrate.
    An empty owner DSN is an error there; for a database the checker does not
    recognise as the bundled one it stays an info (the app DSN may be the
    owner)."""

    BUNDLED = ("postgres://insideout_app:example-app-password"
               "@postgres:5432/insideout")
    BUNDLED_LOCAL = ("postgres://insideout_app:example-app-password"
                     "@127.0.0.1:5442/insideout?sslmode=disable")
    OWNER = ("postgres://insideout_owner:example-owner-password"
             "@postgres:5432/insideout")
    REMOTE = ("postgres://insideout_app:example-app-password"
              "@db.example.org:5432/insideout?sslmode=require")
    OTHER_LOCAL = ("postgres://insideout_app:example-app-password"
                   "@127.0.0.1:5433/insideout")

    def test_bundled_postgres_without_owner_url_fails(self):
        for url in (self.BUNDLED, self.BUNDLED_LOCAL):
            with self.subTest(database_url_host=url.split("@", 1)[1]):
                rc, out = run_check({"DATABASE_URL": url,
                                     "DATABASE_OWNER_URL": ""})
                self.assertEqual(rc, 1, out)
                self.assertIn("FAIL: DATABASE_OWNER_URL", out)
                self.assertNotIn("example-app-password", out)
                self.assertNotIn("example-owner-password", out)

    def test_bundled_postgres_with_owner_url_passes(self):
        rc, out = run_check({"DATABASE_URL": self.BUNDLED,
                             "DATABASE_OWNER_URL": self.OWNER})
        self.assertEqual(rc, 0, out)
        self.assertIn("ok: DATABASE_OWNER_URL", out)
        self.assertNotIn("FAIL", out)

    def test_other_database_without_owner_url_stays_an_info(self):
        for url in (self.REMOTE, self.OTHER_LOCAL):
            with self.subTest(database_url_host=url.split("@", 1)[1]):
                rc, out = run_check({"DATABASE_URL": url,
                                     "DATABASE_OWNER_URL": ""})
                self.assertEqual(rc, 0, out)
                self.assertIn("info: DATABASE_OWNER_URL", out)
                self.assertNotIn("FAIL", out)


if __name__ == "__main__":
    unittest.main()
