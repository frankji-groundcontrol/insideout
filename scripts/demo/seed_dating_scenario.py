#!/usr/bin/env python3
"""Seed the "Spark" dating-app demo story through the InsideOut HTTP API only.

Usage: python3 -I scripts/demo/seed_dating_scenario.py --api http://127.0.0.1:8080/api/v1 \
           [--origin http://localhost:5173] [--password ...] [--admin-email ...] [--admin-password ...]

Registration is invitation-only by default, so a site admin mints one invite per
person (create that admin once with `insideout promote-admin`; see
docs/usage/demo-data.md). Prints one line per request with its status, then the
counts, any findings (unexpected statuses or wrong answers) and the logins.
Refuses when Maya already has the "Spark Labs" workspace.
"""

import argparse
import datetime as dt
import json
import os
import sys
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I drops the script dir
import spark_story as S  # noqa: E402

TODAY = dt.datetime.now(dt.timezone.utc).replace(hour=17, minute=0, second=0, microsecond=0)


class Api:
    def __init__(self, base, origin):
        self.base, self.origin, self.tok, self.findings = base.rstrip("/"), origin, {}, []

    def call(self, who, method, path, body=None, expect=(200, 201), note="", token=None):
        headers = {"Content-Type": "application/json", "Accept": "application/json"}
        if self.origin:
            headers["Origin"] = self.origin
        bearer = token or self.tok.get(who)
        if bearer:
            headers["Authorization"] = "Bearer " + bearer
        data = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(self.base + path, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=90) as resp:
                status, raw, ctype = resp.status, resp.read(), resp.headers.get("Content-Type", "")
        except urllib.error.HTTPError as e:
            status, raw, ctype = e.code, e.read(), e.headers.get("Content-Type", "")
        out = json.loads(raw) if raw and "json" in ctype else raw.decode("utf-8", "replace")
        ok = status in expect
        code = f" [{out.get('code')}]" if status >= 400 and isinstance(out, dict) else ""  # never echo an invite code
        print(f"{'  ' if ok else '!!'} {status} {who:<6} {method:<6} {path.split('?')[0]}{code}  {note}")
        if not ok:
            self.findings.append(f"{method} {path} as {who}: expected {expect}, got {status} {str(out)[:200]}")
        return status, out

    def must(self, *a, **kw):
        status, out = self.call(*a, **kw)
        if status not in kw.get("expect", (200, 201)):
            self.report()
            sys.exit(f"stopping: the story cannot continue after {a[1]} {a[2]} answered {status}")
        return out

    def check(self, cond, finding):
        if not cond:
            print("!! check failed: " + finding)
            self.findings.append(finding)

    def report(self):
        print(f"\nFindings ({len(self.findings)}):")
        for f in self.findings:
            print("  - " + f)


def iso(days):
    return (TODAY + dt.timedelta(days=days)).isoformat().replace("+00:00", "Z")


def people_in(api, args):
    """Log in each person, or mint an invite (as the site admin) and register."""
    ids = {}
    for key, name, _, _ in S.PEOPLE:
        st, out = api.call(key, "POST", "/auth/login", {"email": S.email(key), "password": args.password},
                           expect=(200, 401), note="existing account?")
        if st != 200:
            if "admin" not in api.tok:
                api.tok["admin"] = api.must("admin", "POST", "/auth/login",
                                            {"email": args.admin_email, "password": args.admin_password},
                                            expect=(200,), note="site admin")["accessToken"]
            inv = api.must("admin", "POST", "/admin/invitations", {"email": S.email(key)}, expect=(201,),
                           note=f"invite {name}")
            out = api.must(key, "POST", "/auth/register",
                           {"email": S.email(key), "password": args.password, "username": name,
                            "inviteCode": inv["code"]}, expect=(201,), note=f"register {name}")
        api.tok[key], ids[key] = out["accessToken"], out["id"]
        if key == S.CREATOR:
            spaces = api.must(key, "GET", "/workspaces", expect=(200,))
            if any(w["title"] == S.WORKSPACE for w in spaces):
                sys.exit(f'"{S.WORKSPACE}" already exists for {S.email(key)}: this database is already seeded. '
                         "Run against a fresh database.")
    return ids


def write_prd(api, wid):
    maya = S.CREATOR
    for author, title, content in S.IDEAS:
        idea = api.must(author, "POST", f"/workspaces/{wid}/ideas", {"title": title, "content": content},
                        expect=(201,), note="idea")
        if title == S.IDEAS[0][1]:
            main = idea["id"]
    seen = api.must("leo", "GET", f"/workspaces/{wid}/ideas", expect=(200,), note="ideas Leo can see")
    api.check(all(i["authorId"] == seen[0]["authorId"] for i in seen) and len(seen) == 1,
              f"GET /workspaces/{{id}}/ideas as Leo lists {len(seen)} ideas; ideas are private by default")
    conv = api.must(maya, "POST", f"/ideas/{main}/convert", {"locale": "en-US"}, expect=(201,), note="idea -> PRD")
    prd = conv["prdId"]
    api.must(maya, "PATCH", f"/prds/{prd}", {"title": S.PRD_TITLE, "sections": S.PRD_SECTIONS}, expect=(200,),
             note="write all sections")
    api.must(maya, "POST", f"/prds/{prd}/revisions", {"note": S.REVISION_NOTES[0]}, expect=(201,), note="save revision")
    st, stream = api.call(maya, "POST", f"/conversations/{conv['conversationId']}/messages",
                          {"content": S.COACH_MESSAGE}, expect=(200,), note="coach turn (SSE)")
    if st == 200:
        print(f"     coach stream: {str(stream).count('data:')} data lines")
    goals = S.PRD_SECTIONS["goals"] + " 4. No member sees more than three new matches a day."
    api.must(maya, "PATCH", f"/prds/{prd}", {"sections": {"goals": goals}}, expect=(200,), note="edit goals")
    api.must(maya, "POST", f"/prds/{prd}/revisions", {"note": S.REVISION_NOTES[1]}, expect=(201,), note="save revision")
    api.must(maya, "POST", f"/prds/{prd}/status", {"status": "reviewing"}, expect=(200,), note="submit for review")
    return prd


def build_roadmap(api, prd):
    draft = api.must(S.CREATOR, "POST", f"/prds/{prd}/build", {}, expect=(200,), note="draft roadmap")["tree"]
    size = lambda n: 1 + sum(size(c) for c in n.get("children") or [])  # noqa: E731
    count = size(draft)
    print(f"     template draft: {count} nodes; confirming the story tree instead")

    def plan(n):
        return {"title": n["title"], "children": [plan(c) for c in n.get("children", [])]}
    tree = {"title": S.PROJECT_ROOT, "description": S.PRD_TITLE, "children": [plan(n) for n in S.TREE]}
    out = api.must(S.CREATOR, "POST", f"/prds/{prd}/build/confirm", {"tree": tree}, expect=(201,), note="confirm build")
    return out["projectId"]


def nodes_by_title(api, pid):
    return {n["title"]: n for n in api.must(S.CREATOR, "GET", f"/projects/{pid}/roadmap", expect=(200,))}


def run_tree(api, pid, ids, uid_of):
    nodes = nodes_by_title(api, pid)
    n = lambda t: nodes[t]["id"]  # noqa: E731
    counts = {"delegations": 0, "evidence": 0}
    parents = {v["parentId"] for v in nodes.values()}
    seeded = [t for t, v in nodes.items() if v["status"] == "in_progress" and v["id"] in parents]
    api.check(not seeded, f"POST /prds/{{id}}/build/confirm seeded a branch, not a leaf, in progress: {seeded}")
    owner_of = {}  # title -> person who acts on it (nearest owner, else the creator)
    for parent, node in S.walk():
        inherited = owner_of.get(parent, S.CREATOR)
        if node.get("owner"):
            api.must(inherited, "POST", f"/roadmap/{n(node['title'])}/delegate",
                     {"toUserId": ids[node["owner"]], "note": f"{node['title']} is yours"}, expect=(200,),
                     note=f"{inherited} -> {node['owner']}: {node['title']}")
            counts["delegations"] += 1
        owner_of[node["title"]] = node.get("owner") or inherited
    # A delegate may re-delegate descendants, never the node handed to them.
    api.call("leo", "POST", f"/roadmap/{n('Matching')}/delegate", {"toUserId": ids["sam"], "note": "try"},
             expect=(403,), note="Leo re-delegates the node he was handed (refused)")
    for _, node in S.walk():
        if "status" not in node:
            continue
        who, nid, want = owner_of[node["title"]], n(node["title"]), node["status"]
        body = {"deadline": iso(node["due"]), "acceptance": node["acceptance"]}
        if nodes[node["title"]]["status"] == "locked":
            api.must(who, "PATCH", f"/roadmap/{nid}", {"status": "pending"}, expect=(200,), note="unlock")
        if want != "pending":
            body["status"] = "in_progress"
        api.must(who, "PATCH", f"/roadmap/{nid}", body, expect=(200,), note=f"{node['title']}: deadline, acceptance")
        for detail, url in node.get("evidence", []):
            api.must(who, "POST", f"/roadmap/{nid}/evidence",
                     {"detail": detail, "sourceUrl": url, "observedAt": TODAY.date().isoformat()},
                     expect=(201,), note="evidence (" + ("link" if url else "note") + ")")
            counts["evidence"] += 1
        if want in ("done", "validated"):
            api.must(who, "PATCH", f"/roadmap/{nid}", {"status": "done"}, expect=(200,), note="done")
        if want == "validated":
            out = api.must(S.CREATOR, "POST", f"/roadmap/{nid}/validate", {"reason": node["validation"]},
                           expect=(200,), note="validate (human)")
            api.check(out["status"] == "validated", f"POST /roadmap/{{nid}}/validate left status {out['status']}")
    mine = api.must("sam", "GET", f"/workspaces/{uid_of['wid']}/roadmap/mine", expect=(200,), note="Sam's tree")
    sam_roots = sorted(m["node"]["title"] for m in mine)
    print(f"     Sam owns: {sam_roots}")
    return nodes, counts, owner_of


def proposals_and_handback(api, pid, nodes, owner_of):
    for author, about, kind, summary, items, hint, decision in S.PROPOSALS:
        body = {"projectId": pid, "kind": kind, "summary": summary, "nodeId": nodes[about]["id"],
                "items": [{"action": "add_node", "title": t, "parentHint": hint} for t in items]}
        prop = api.must(author, "POST", "/agent/propose", body, expect=(201,), note=f"proposal by {author}")
        out = api.must(S.CREATOR, "POST", f"/agent/proposals/{prop['id']}/decision",
                       {"decision": decision, "reason": f"{decision} by the project owner",
                        "apply": decision == "accepted"}, expect=(200,), note=decision)
        api.check(len(out["appliedNodes"]) == (len(items) if decision == "accepted" else 0),
                  f"proposal decision '{decision}' applied {len(out['appliedNodes'])} nodes")
    handed = [nd["title"] for _, nd in S.walk() if nd.get("handback")]
    for title in handed:
        out = api.must(owner_of[title], "POST", f"/roadmap/{nodes[title]['id']}/handback",
                       {"note": "Marketing will own the listing"}, expect=(200,), note=f"hand back {title}")
        api.check(out.get("ownerId") is not None, f"handback of '{title}' left no owner")


def updates_feed(api, pid):
    for author, kind, content in S.UPDATES:
        api.must(author, "POST", f"/projects/{pid}/updates", {"kind": kind, "content": content}, expect=(201,),
                 note=kind)
    total, pages, cursor = 0, 0, None
    while True:
        q = f"/projects/{pid}/updates?limit=10" + (f"&before={cursor}" if cursor else "")
        page = api.must(S.CREATOR, "GET", q, expect=(200,), note=f"feed page {pages + 1}")
        total, pages, cursor = total + len(page["updates"]), pages + 1, page["nextCursor"]
        if not cursor:
            return total, pages


def agent_token(api, pid, prd, nodes):
    root = nodes["Matching"]["id"]
    out = api.must("leo", "POST", "/agent/token", {"rootNodeId": root}, expect=(200, 201), note="agent token (Matching)")
    tok = out["accessToken"]
    api.check(out.get("rootNodeId") == root, "POST /agent/token did not echo rootNodeId")
    api.call("agent", "POST", "/agent/checkpoint",
             {"projectId": pid, "nodeId": nodes["Daily match limit"]["id"],
              "summary": "Daily limit job deployed to staging; checking counts tonight."},
             expect=(201,), token=tok, note="checkpoint")
    api.call("agent", "PATCH", f"/roadmap/{nodes['Daily drop UI']['id']}",
             {"description": "Reveal animation spec linked from the design file."},
             expect=(200,), token=tok, note="edit inside its root")
    refusals = {}
    for label, method, path, body in [
        ("outside root", "PATCH", f"/roadmap/{nodes['Conversation starters from prompts']['id']}",
         {"description": "agent edit"}),
        ("human-only: delegate", "POST", f"/roadmap/{nodes['Daily drop UI']['id']}/delegate",
         {"toUserId": nodes["Matching"]["ownerId"] or "", "note": "agent"}),
        ("human-only: commit", "POST", f"/prds/{prd}/commit",
         {"name": "agent commit", "primaryAudience": "decision"}),
    ]:
        st, res = api.call("agent", method, path, body, expect=(403,), token=tok, note=label)
        refusals[label] = f"{st} {res.get('code') if isinstance(res, dict) else ''}"
    return refusals


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--api", default=os.environ.get("INSIDEOUT_API", "http://127.0.0.1:8080/api/v1"))
    ap.add_argument("--origin", default=os.environ.get("INSIDEOUT_ORIGIN", ""))
    ap.add_argument("--password", default="spark-demo-2026", help="password for all five people (dev default)")
    ap.add_argument("--admin-email", default=f"ops@{S.DOMAIN}", help="an existing site admin (promote-admin)")
    ap.add_argument("--admin-password", default=None, help="defaults to --password")
    args = ap.parse_args()
    args.admin_password = args.admin_password or args.password
    api = Api(args.api, args.origin)
    print("signup mode:", api.must("anon", "GET", "/auth/config", expect=(200,)).get("signupMode"))

    ids = people_in(api, args)
    ws = api.must(S.CREATOR, "POST", "/workspaces", {"title": S.WORKSPACE,
                  "description": "A small team building Spark, a dating app for slow, intentional matching."},
                  expect=(201,), note="create workspace")
    wid, code = ws["id"], ws["code"]
    for key, *_ in S.PEOPLE[1:]:
        api.must(key, "POST", "/workspaces/join", {"code": code}, expect=(200,), note="join with code")

    prd = write_prd(api, wid)
    api.must(S.CREATOR, "PATCH", f"/workspaces/{wid}/members/{ids[S.PROMOTED]}", {"role": "admin"},
             expect=(200,), note="promote Jo to admin")
    api.must(S.PROMOTED, "POST", f"/prds/{prd}/status", {"status": "approved"}, expect=(200,), note="Jo approves")
    api.must(S.CREATOR, "POST", f"/prds/{prd}/commit", S.COMMIT, expect=(201,), note="commit version (human)")
    revisions = len(api.must(S.CREATOR, "GET", f"/prds/{prd}/revisions", expect=(200,)))

    pid = build_roadmap(api, prd)
    project = api.must(S.CREATOR, "GET", f"/projects/{pid}", expect=(200,))
    api.check(project.get("ownerId") == ids[S.CREATOR], "the built project is not owned by Maya")
    nodes, counts, owner_of = run_tree(api, pid, ids, {"wid": wid})
    proposals_and_handback(api, pid, nodes, owner_of)
    nodes = nodes_by_title(api, pid)
    total, pages = updates_feed(api, pid)
    refusals = agent_token(api, pid, prd, nodes)

    members = api.must(S.CREATOR, "GET", f"/workspaces/{wid}/members", expect=(200,))
    print(f"\nBuilt: {len(S.PEOPLE)} users, {len(S.IDEAS)} ideas, {revisions} PRD revisions, 1 commit, "
          f"{len(nodes)} roadmap nodes, {counts['delegations']} delegations (+1 hand-back), "
          f"{counts['evidence']} evidence rows, {len(S.PROPOSALS)} proposals, "
          f"{total} feed rows over {pages} pages ({len(S.UPDATES)} posted by people)")
    print("Agent token refusals: " + "; ".join(f"{k}: {v}" for k, v in refusals.items()))
    api.report()
    print(f'\nLogins (password: the --password value) for workspace "{S.WORKSPACE}", join code {code}:')
    roles = {m["userId"]: m["role"] for m in members}
    for key, name, title, _ in S.PEOPLE:
        print(f"  {S.email(key):<28} {name:<11} {title:<22} {roles.get(ids[key], '?')}")
    return 1 if api.findings else 0


if __name__ == "__main__":
    sys.exit(main())
