#!/usr/bin/env python3
"""The Spark demo story table is well formed — run: python3 -I scripts/tests/test_demo_story.py

No server and no network: it checks scripts/demo/spark_story.py against the
rules the seed script relies on (unique titles for lookups, a leaf state for
every task, delegations a delegate is allowed to make, PRD keys the server
accepts) so a table edit fails here instead of halfway through a seed run.
"""

import re
import sys
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / "scripts" / "demo"))
import spark_story as S  # noqa: E402
import seed_dating_scenario  # noqa: E402,F401  (imports cleanly, runs nothing)

KEYS = {k for k, *_ in S.PEOPLE}


def owners_on_path():
    """title -> (own owner or None, nearest owner above it or the creator)."""
    out, acting = {}, {}
    for parent, node in S.walk():
        above = acting.get(parent, S.CREATOR)
        out[node["title"]] = (node.get("owner"), above)
        acting[node["title"]] = node.get("owner") or above
    return out, acting


class StoryTable(unittest.TestCase):
    def test_people(self):
        self.assertEqual(len(KEYS), len(S.PEOPLE), "person keys are unique")
        self.assertEqual(len(S.PEOPLE), 5)
        self.assertIn(S.CREATOR, KEYS)
        self.assertIn(S.PROMOTED, KEYS)
        for key, name, title, role in S.PEOPLE:
            self.assertRegex(S.email(key), r"^[a-z]+@[a-z.]+\.test$", "reserved .test addresses only")
            self.assertIn(role, ("admin", "member"))
            self.assertTrue(name and title)

    def test_ideas_and_prd(self):
        self.assertTrue({a for a, *_ in S.IDEAS} >= KEYS, "every person captures an idea")
        self.assertEqual(S.IDEAS[0][0], S.CREATOR, "the creator converts the first idea")
        go = (REPO / "server/internal/store/prds.go").read_text()
        block = re.search(r"PrdSectionKeys = \[\]string\{(.*?)\}", go, re.S).group(1)
        server_keys = set(re.findall(r'"(\w+)"', block))
        self.assertTrue(set(S.PRD_SECTIONS) <= server_keys, set(S.PRD_SECTIONS) - server_keys)
        for key in ("background", "users", "goals", "nonGoals", "requirements"):
            self.assertTrue(S.PRD_SECTIONS[key].strip())
        self.assertIn(S.COMMIT["primaryAudience"], ("decision", "management", "delivery", "validation"))

    def test_tree_shape(self):
        titles = [n["title"] for _, n in S.walk()]
        self.assertEqual(len(titles), len(set(titles)), "titles are unique (lookups and parentHint)")
        self.assertNotIn(S.PROJECT_ROOT, titles)
        self.assertEqual([b["title"] for b in S.TREE],
                         ["Onboarding & profiles", "Matching", "Chat", "Safety & trust", "Launch"])
        for branch in S.TREE:
            self.assertTrue(3 <= len(branch["children"]) <= 6, branch["title"])
        self.assertTrue(any("children" in c for b in S.TREE for c in b["children"]), "some tasks two levels deep")

    def test_leaves_and_branches(self):
        seen, dues = set(), []
        for _, node in S.walk():
            if "children" in node:
                for field in ("status", "due", "evidence"):
                    self.assertNotIn(field, node, f"branch {node['title']} carries no {field}")
                continue
            self.assertIn(node["status"], S.STATUSES, node["title"])
            self.assertIsInstance(node["due"], int)
            self.assertTrue(node["acceptance"].strip())
            finished = node["status"] in ("done", "validated")
            self.assertEqual(finished, bool(node.get("evidence")), f"evidence on finished tasks: {node['title']}")
            self.assertEqual(node["status"] == "validated", "validation" in node)
            seen.add(node["status"])
            dues.append(node["due"])
        self.assertEqual(seen, set(S.STATUSES), "statuses spread over all four")
        self.assertTrue(any(d < 0 for d in dues) and any(0 <= d <= 7 for d in dues) and any(d >= 28 for d in dues),
                        "deadlines past due, this week and next month")
        kinds = {bool(url) for _, n in S.walk() for _, url in n.get("evidence", [])}
        self.assertEqual(kinds, {True, False}, "evidence of both link and note kinds")

    def test_delegations(self):
        paths, _ = owners_on_path()
        top = {b["title"]: b.get("owner") for b in S.TREE}
        self.assertEqual(top["Matching"], "leo")
        self.assertEqual(top["Chat"], "leo")
        self.assertEqual(top["Onboarding & profiles"], "priya")
        self.assertEqual(top["Safety & trust"], "jo")
        by_leo = {}
        for (branch_title, node) in ((b["title"], n) for b in S.TREE for _, n in S.walk(b.get("children", []))):
            own, above = paths[node["title"]]
            if own:
                self.assertIn(own, KEYS)
                self.assertNotEqual(own, above, f"{node['title']} is delegated to someone new")
                if above == "leo":
                    by_leo.setdefault(branch_title, []).append(own)
        self.assertEqual(by_leo, {"Matching": ["sam", "sam"], "Chat": ["sam"]})
        handbacks = [n for _, n in S.walk() if n.get("handback")]
        self.assertEqual(len(handbacks), 1)
        self.assertEqual(paths[handbacks[0]["title"]], ("priya", S.CREATOR), "Priya hands a task back to Maya")

    def test_proposals_and_updates(self):
        _, acting = owners_on_path()
        titles = {n["title"] for _, n in S.walk()}
        self.assertEqual(sorted(p[-1] for p in S.PROPOSALS), ["accepted", "rejected"])
        for author, about, kind, summary, items, hint, _ in S.PROPOSALS:
            self.assertEqual(acting[about], author, "a proposal stays inside its author's subtree")
            self.assertIn(kind, ("structure", "scope", "priority"))
            self.assertIn(hint, titles)
            self.assertFalse(set(items) & titles, "proposed nodes are new")
            self.assertTrue(summary.strip())
        self.assertTrue(8 <= len(S.UPDATES) <= 12)
        self.assertTrue({k for _, k, _ in S.UPDATES} <= {"progress", "blocker", "note"})
        self.assertIn("blocker", {k for _, k, _ in S.UPDATES})
        self.assertGreaterEqual(len({a for a, _, _ in S.UPDATES}), 4)


if __name__ == "__main__":
    unittest.main()
