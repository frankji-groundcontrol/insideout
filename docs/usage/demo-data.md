# Demo data: the Spark dating-app story

`scripts/demo/seed_dating_scenario.py` fills a **fresh, disposable** database
with a realistic team story by calling the HTTP API only (no database
writes). It is a demo seed and an end-to-end walk of the product: every
request prints one line with its status, and any unexpected status or wrong
answer is listed as a finding at the end (exit code 1).

## Run it

Registration is invitation-only by default, so the script needs one site admin
to mint the invitations. Create that admin once with the server's operator
subcommand (owner DSN; the password is read from stdin):

```bash
printf '%s\n' '<admin-password>' | insideout promote-admin ops@spark.example.test
```

Then point the script at the API:

```bash
python3 -I scripts/demo/seed_dating_scenario.py \
  --api http://127.0.0.1:<api-port>/api/v1 \
  --origin http://localhost:<web-port> \
  --admin-password '<admin-password>'
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--api` | `$INSIDEOUT_API`, else `http://127.0.0.1:8080/api/v1` | API base |
| `--origin` | `$INSIDEOUT_ORIGIN`, else none | `Origin` header, when CORS needs one |
| `--password` | `spark-demo-2026` (dev default) | Password for all five people |
| `--admin-email` | `ops@spark.example.test` | An existing site admin |
| `--admin-password` | the `--password` value | That admin's password |

A second run refuses politely: when Maya already has the "Spark Labs"
workspace it prints that the database is already seeded and exits 1. Use a
fresh database to seed again. Never point it at a shared or production
server.

The story itself lives in `scripts/demo/spark_story.py`;
`python3 -I scripts/tests/test_demo_story.py` checks that table without a
server.

## What it builds

- **People**: five accounts, each invited by the admin and registered through
  `POST /auth/register`. Maya creates the "Spark Labs" workspace; the others
  join with its code. Maya later promotes Jo to workspace admin.
- **Ideas and PRD**: six ideas (private to their authors). Maya converts the
  main one into a PRD, writes its sections, runs one coach turn (without an
  LLM key the coach echoes), saves two revisions and submits it. Jo approves
  it, and Maya commits the "Spark v1 baseline" version.
- **Roadmap**: the template planner drafts a tree; the script confirms its own
  33-node tree instead: a "Spark MVP" root, five branches (Onboarding &
  profiles, Matching, Chat, Safety & trust, Launch) and three to five tasks
  each, some two levels deep, with deadlines (past due, this week, next
  month), acceptance sentences and statuses spread over pending, in progress,
  done and validated.
- **Collaboration**: Maya delegates four branches and one Launch task; Leo
  re-delegates two Matching tasks and one Chat task to Sam (his own attempt
  to re-delegate "Matching" is refused, as it should be). Done tasks carry link
  and note evidence; Maya validates "Block and hide". Sam's proposal inside
  his subtree is accepted and applied; Jo's is rejected. Priya hands the App
  Store task back to Maya. Ten progress, blocker and note updates come from
  different people, and the feed is read page by page.
- **Agent token**: Leo mints a token scoped to "Matching". It posts a
  checkpoint and edits a node inside its root; it is refused (403
  `agent_scope_denied`) outside the root and on human-only routes (delegate,
  commit).

## Logins

All five use the `--password` value; the script prints the workspace join
code at the end.

| Email | Person | Role |
| --- | --- | --- |
| `maya@spark.example.test` | Maya Chen, product lead | workspace admin, project owner |
| `leo@spark.example.test` | Leo Park, engineering lead | member |
| `priya@spark.example.test` | Priya Nair, product designer | member |
| `sam@spark.example.test` | Sam Okafor, backend engineer | member |
| `jo@spark.example.test` | Jo Rivera, QA and trust & safety | workspace admin (promoted) |
