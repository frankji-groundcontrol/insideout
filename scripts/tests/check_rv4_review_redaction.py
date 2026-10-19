#!/usr/bin/env python3
"""Pin the whole-change review repairs of the redaction and ledger guards
(rv4-FR-F45, F46, F53, F55).

Usage: python3 -I scripts/tests/check_rv4_review_redaction.py

Stdlib only. Each class reproduces a finding of the review of
d53035a..rv4/integration against the real guard in a throwaway repository, so
the proof does not depend on how this checkout happens to look:

  LedgerHistoryArm  F45  the ledger-honesty history arm in a shallow clone
  ProviderIdArm     F46  the UUID-shaped provider-id scan beyond three named records
"""
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest
from collections import Counter

ROOT = pathlib.Path(__file__).resolve().parents[2]
AT = "@"  # addresses are built around AT so this file never spells one itself


def env(**extra):
    """A clean environment: no GIT_* leaked by a hook, no CI unless a case sets it."""
    base = {k: v for k, v in os.environ.items() if not k.startswith("GIT_") and k != "CI"}
    base.update(extra)
    return base


def git(args, cwd):
    # Throwaway repos: no signing, no hooks, a fixed identity.
    cmd = ["git", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null",
           "-c", "user.name=probe", "-c", f"user.email=probe{AT}example.test", *args]
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=env())
    if r.returncode:
        raise RuntimeError(f"git {args[0]} failed: {r.stderr.strip()}")
    return r.stdout.strip()


def commit(repo, files, message):
    for name, text in files.items():
        path = pathlib.Path(repo, name)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
    git(["add", "--", *files], repo)
    git(["commit", "-q", "-m", message], repo)
    return git(["rev-parse", "HEAD"], repo)


LEDGER_GUARD = "scripts/tests/check_ledger_honesty.py"
LEDGER = "docs/review/2026-10-10-disposition-ledger/04-multirole-3.md"


def ledger(rows):
    """A ledger the guard parses: rows are (id, status, cited fix sha)."""
    tally = Counter(status for _, status, _ in rows)
    counts = " + ".join(f"{n} {status}" for status, n in tally.items())
    lines = ["# Probe ledger", ""]
    lines += [f"| {id_} | M | {status} | fix {sha} |" for id_, status, sha in rows]
    return "\n".join(lines + ["", f"Counts of the probe: {len(rows)} rows = {counts}.", ""])


class LedgerHistoryArm(unittest.TestCase):
    """rv4-FR-F45: the history arm skips every row whose commit git cannot see,
    and the CI checkout is depth 1, so there it checked nothing while the
    success line still said the labels match their diffs."""

    def clones(self, tmp, labels=("Fixed", "Recorded")):
        """A full clone and a depth-1 clone of an origin whose tip carries the
        guard and a ledger citing the two older commits (code, then docs only)."""
        origin = pathlib.Path(tmp, "origin")
        origin.mkdir()
        git(["init", "-q"], origin)
        code = commit(origin, {"src/app.go": "package app\n"}, "code")
        docs = commit(origin, {"docs/note.md": "note\n"}, "docs only")
        guard = (ROOT / LEDGER_GUARD).read_text(encoding="utf-8")
        commit(origin, {LEDGER_GUARD: guard,
                        LEDGER: ledger([("T-1", labels[0], code), ("T-2", labels[1], docs)])}, "ledger")
        full, shallow = pathlib.Path(tmp, "full"), pathlib.Path(tmp, "shallow")
        git(["clone", "-q", str(origin), str(full)], tmp)
        git(["clone", "-q", "--depth", "1", origin.as_uri(), str(shallow)], tmp)
        return full, shallow

    def integrated_clone(self, tmp, labels=("Fixed", "Recorded")):
        """What a full CI checkout holds: the ledger cites worker-branch commits,
        the worker branch was never pushed, and main carries `cherry-pick -x`
        copies of them. Cloned over file:// so unreachable objects stay behind."""
        origin = pathlib.Path(tmp, "origin")
        origin.mkdir()
        git(["init", "-q", "-b", "main"], origin)
        commit(origin, {"README.md": "base\n"}, "base")
        git(["checkout", "-q", "-b", "worker"], origin)
        code = commit(origin, {"src/app.go": "package app\n"}, "code")
        docs = commit(origin, {"docs/note.md": "note\n"}, "docs only")
        git(["checkout", "-q", "main"], origin)
        git(["cherry-pick", "-x", code, docs], origin)
        guard = (ROOT / LEDGER_GUARD).read_text(encoding="utf-8")
        commit(origin, {LEDGER_GUARD: guard,
                        LEDGER: ledger([("T-1", labels[0], code), ("T-2", labels[1], docs)])}, "ledger")
        git(["branch", "-q", "-D", "worker"], origin)
        clone = pathlib.Path(tmp, "ci")
        git(["clone", "-q", origin.as_uri(), str(clone)], tmp)
        for sha in (code, docs):  # the premise: this checkout cannot see them directly
            with self.assertRaises(RuntimeError):
                git(["cat-file", "-e", sha + "^{commit}"], clone)
        return clone

    def guard(self, repo, ci=False):
        extra = {"CI": "1"} if ci else {}
        return subprocess.run([sys.executable, "-I", LEDGER_GUARD], cwd=repo,
                              capture_output=True, text=True, env=env(**extra))

    def test_a_worker_branch_id_is_checked_through_its_cherry_pick_copy(self):
        # The ledger cites pre-integration ids; only the -x copies reach a CI clone.
        # Without this, "fail in CI when a cited commit is unseen" is red forever.
        with tempfile.TemporaryDirectory() as tmp:
            clone = self.integrated_clone(tmp)
            for ci in (False, True):
                with self.subTest(ci=ci):
                    r = self.guard(clone, ci)
                    self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
                    self.assertIn("2 of 2 cited fix commits visible to git, labels match their diffs", r.stdout)

    def test_a_mislabelled_row_is_caught_through_its_cherry_pick_copy(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = self.integrated_clone(tmp, labels=("Recorded", "Fixed"))
            r = self.guard(clone, ci=True)
            self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
            self.assertIn("T-1 is Recorded but", r.stdout)
            self.assertIn("T-2 is Fixed but", r.stdout)

    def test_full_clone_checks_the_labels(self):
        with tempfile.TemporaryDirectory() as tmp:
            full, _ = self.clones(tmp)
            for ci in (False, True):
                with self.subTest(ci=ci):
                    r = self.guard(full, ci)
                    self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
                    self.assertIn("2 of 2 cited fix commits visible to git, labels match their diffs", r.stdout)

    def test_full_clone_still_catches_a_mislabelled_row(self):
        # The arm itself, kept honest: code labelled Recorded, docs-only labelled Fixed.
        with tempfile.TemporaryDirectory() as tmp:
            full, _ = self.clones(tmp, labels=("Recorded", "Fixed"))
            r = self.guard(full)
            self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
            self.assertIn("T-1 is Recorded but", r.stdout)
            self.assertIn("T-2 is Fixed but", r.stdout)

    def test_shallow_clone_does_not_claim_the_labels_were_checked(self):
        with tempfile.TemporaryDirectory() as tmp:
            _, shallow = self.clones(tmp)
            r = self.guard(shallow)
            self.assertEqual(r.returncode, 0, "a developer's shallow clone is not an error: " + r.stdout)
            self.assertNotIn("labels match their diffs", r.stdout)
            self.assertIn("history arm NOT run", r.stdout)

    def test_shallow_clone_under_ci_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            _, shallow = self.clones(tmp)
            r = self.guard(shallow, ci=True)
            self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
            self.assertIn("fetch-depth: 0", r.stdout)


class ProviderIdArm(unittest.TestCase):
    """rv4-FR-F46: the provider-id scan of check_redaction_addresses.py read three
    named records, so the next dated record to paste a deployment id passed every
    guard. Driven black-box: a throwaway repo holds a copy of that test and the
    records the old arm named; one unlisted record gets an id planted in it."""

    TEST = "TrackedTree.test_dated_records_carry_no_provider_ids"
    NAMED = ("docs/changelogs/2026-08-18-flutter-web-host.md",
             "docs/changelogs/2026-08-21-dogfood-and-cleanup.md",
             "docs/plans/2026-09-18-cobalt-flutter-goal.md")
    # UUID-shaped but synthetic, built from parts so this file spells none.
    ID = "-".join(("3f6c1b52", "8d1e", "4a7b", "9c0d", "5e2f7a9b1c34"))
    NIL = "-".join(("00000000", "0000", "0000", "0000", "000000000000"))

    def run_arm(self, extra):
        with tempfile.TemporaryDirectory() as tmp:
            files = {n: "clean\n" for n in self.NAMED}
            files.update(extra)
            files["scripts/tests/check_redaction_addresses.py"] = \
                (ROOT / "scripts/tests/check_redaction_addresses.py").read_text(encoding="utf-8")
            git(["init", "-q"], tmp)
            for name, text in files.items():
                path = pathlib.Path(tmp, name)
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(text, encoding="utf-8")
            git(["add", "--", *files], tmp)
            return subprocess.run([sys.executable, "-I", "scripts/tests/check_redaction_addresses.py", self.TEST],
                                  cwd=tmp, capture_output=True, text=True, env=env())

    def test_a_clean_snapshot_passes(self):
        r = self.run_arm({})
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)

    def test_an_id_in_any_tracked_record_under_docs_fails_without_being_echoed(self):
        for path in ("docs/changelogs/2026-10-19-probe.md", "docs/plans/2026-10-19-probe.track.yaml",
                     "docs/usage/probe.md"):
            with self.subTest(path=path):
                r = self.run_arm({path: f"first\nsecond\nDeploy {self.ID} (path strategy)\n"})
                out = r.stdout + r.stderr
                self.assertNotEqual(r.returncode, 0, "an unlisted record's provider id passed the arm")
                self.assertIn(f"{path}:3", out)
                self.assertNotIn(self.ID, out)

    def test_the_nil_uuid_is_not_a_provider_id(self):
        r = self.run_arm({"docs/learning/probe.md": f"NULLIF($3::uuid, '{self.NIL}'::uuid)\n"})
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)


if __name__ == "__main__":
    unittest.main()
