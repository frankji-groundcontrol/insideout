#!/usr/bin/env python3
"""rv4-CLOSE-04 - a disposition-ledger row labelled Fixed must have changed
something that runs.

Review 4 found the review-3 ledger counting eleven rows as "closed by code"
whose cited fix commit touched only docs/. Recording a gap is a legitimate
disposition, but it is Recorded, not Fixed. Two arms, both read from the
ledger table itself:

  history - each `fix <sha>` a row cites is looked up with git. A Fixed row
            whose fix commits all changed only files under docs/ fails, and a
            Recorded row whose fix commit changed anything outside docs/
            fails, so the label matches the diff in both directions. The
            ledger cites pre-integration commit ids that exist only where the
            worker branches do; a commit git cannot see is skipped and
            counted in the summary, never failed.
  counts  - the ledger's `Counts ...: N rows = a Status + b Status` line must
            state the tally the table has, so the header cannot drift from
            the rows again. This arm needs no git, so it holds on a clone
            with no history.

Run: python3 -I scripts/tests/check_ledger_honesty.py
"""
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
LEDGER = ROOT / "docs/review/2026-10-10-disposition-ledger/04-multirole-3.md"
ROW = re.compile(r"^\| ([A-Z]+-\d+) \| [A-Z] \| (.+?) \| (.*?) \|?$")
COUNTS = re.compile(r"^Counts[^:\n]*: (\d+) rows = (.+)\.$", re.M)


def changed_files(sha):
    """Files the commit changed, or None when git cannot see it here."""
    try:
        r = subprocess.run(["git", "-C", str(ROOT), "show", "--name-only", "--format=", sha + "^{commit}"],
                           capture_output=True, text=True)
    except OSError:
        return None
    files = [f for f in r.stdout.split("\n") if f]
    return files if r.returncode == 0 and files else None


def main():
    if not LEDGER.is_file():
        print("ledger honesty skipped: no review-3 ledger")
        return
    rows = []
    for line in LEDGER.read_text().splitlines():
        m = ROW.match(line)
        if m:
            rows.append((m.group(1), m.group(2), m.group(3)))
    tally = Counter(status for _, status, _ in rows)

    problems = []

    # counts arm
    m = COUNTS.search(LEDGER.read_text())
    if not m:
        problems.append(f"{LEDGER.name} has no 'Counts ...: N rows = a Status + b Status' line; "
                        f"the table has {len(rows)} rows ({dict(tally)})")
    else:
        stated = {}
        for part in m.group(2).split(" + "):
            pm = re.match(r"(\d+) (.+)$", part.strip())
            stated[pm.group(2) if pm else part] = int(pm.group(1)) if pm else -1
        if int(m.group(1)) != len(rows) or stated != dict(tally):
            problems.append(f"{LEDGER.name} Counts line says {m.group(1)} rows = {stated}; "
                            f"the table has {len(rows)} = {dict(tally)}")

    # history arm
    cited = seen = 0
    for id_, status, resolution in rows:
        if status not in ("Fixed", "Recorded"):
            continue
        shas = re.findall(r"\bfix\s+([0-9a-f]{7,40})\b", resolution)
        files = [changed_files(s) for s in shas]
        cited += len(shas)
        seen += sum(f is not None for f in files)
        if not shas or None in files:
            continue  # nothing cited, or a commit this clone cannot see
        outside = [p for f in files for p in f if not p.startswith("docs/")]
        if status == "Fixed" and not outside:
            problems.append(f"{id_} is Fixed but its fix commit(s) {', '.join(shas)} changed only docs/ - label it Recorded")
        if status == "Recorded" and outside:
            problems.append(f"{id_} is Recorded but its fix commit(s) {', '.join(shas)} changed {outside[0]} outside docs/ - label it Fixed")
    if problems:
        print("\n".join("FAIL: " + p for p in problems))
        sys.exit(1)
    print(f"ledger honesty ok: {len(rows)} rows, Counts line matches the table; "
          f"{seen} of {cited} cited fix commits visible to git, labels match their diffs")


if __name__ == "__main__":
    main()
