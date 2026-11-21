# 21 · Code structure: Flutter client, scripts and layout

**Lens:** modularity of the client, the scripts and the repository layout
**Score:** **6.2 / 10**
**Bottom line:** The Flutter client is in reasonable structural shape: feature folders, a per-resource API client, an executable 350-line guard and 156 test files. The scripts are where modularity breaks down: the same job is implemented two to four times and the copies have already diverged. `scripts/setup.sh` writes a pasted database URL unquoted, so a URL with two query parameters is lost when `dev.sh` sources the file (reproduced against a scratch env file), while the Go wizard does it correctly. The key tool carries its own placeholder rule and would record the skeleton's placeholder JWT secret as a real key. The working tree also leaves three contract gates red, not the one the scout recorded, and CI runs all of them. The 350-line rule is being met by trimming to fit rather than by separating responsibilities: two client files sit at 349 lines, one holding a 161-line method that fetches, renders and writes. Not checked: I did not re-run the Go or Flutter test suites, and I did not read the docs index structure, which belongs to another reviewer.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read `client/lib` (136 Dart files, 17,891 lines) and the `client/test` layout (156 files, 21,143 lines); `scripts/` including `scripts/tests`, `scripts/tests/keys` and `scripts/sim`; `server/scripts`; `config/`, `docker/`, `docker-compose.yml`; the five workflows; `hosts/`, `skills/`, `insideout.yaml` and the root markdown files. For comparison I read the Go setup wizard (`server/internal/setup`, `server/cmd/insideout/cli_setup*.go`). I measured line counts and the longest functions, ran all 19 `scripts/tests/check_*.py` gates, and ran `scripts/setup.sh` and `scripts/keys.py` against scratch files only. I did not re-run the Go or Flutter suites and did not review the docs index.

## What is good

- `client/lib/api/client.dart:27` builds one `ApiClient` from per-resource mixins, and the models are split by domain.
- The budget is executable: `client/test/line_budget_test.dart:24` and `server/internal/linebudget/line_budget_test.go:27`. No client file exceeds 350 lines.
- `scripts/env_catalog.py` keeps decisions out of the curses shell and reads the variable list from `.env.example`; `server/internal/setup/catalog/parity_test.go` diffs the Go and Python readers.
- `.github/workflows/ci.yml:347` runs every gate through a glob.
- Both languages carry 403 translation keys in four tables behind `client/lib/l10n/en.dart`.

## Findings, worst first

**STRC-01 (High)** `scripts/tests/check_first_run_copy.py:190`: `if "api_base=" not in install:`. Three gates are red in the working tree: this one (exit 2, the uncommitted `scripts/install.sh:73` dropped that line), `check_supply_chain.py` (exit 1, lines 51-52 still list two modules the uncommitted `go.mod` dropped), and `check_docs_truth_rv4.py`. Fix: update the anchor, delete the two entries, name the two files in `backend.md`.

**STRC-02 (High)** `scripts/setup.sh:143`: `out.append(prefix + value + nl)`. A third local-server installer with a private, unquoted, non-atomic `.env` writer. Reproduced on a scratch file: a database URL with two query parameters is written unquoted, bash sourcing leaves `DATABASE_URL` unset, and `env.sh check` fails. `scripts/install.sh:84,91` and `README.md:105` still route to it, and no test executes it. Fix: make it a shim over `insideout setup`.

**STRC-03 (Medium)** `scripts/keys.py:32`: `PLACEHOLDER = re.compile(r"changeme|...`. The skeleton marker is `change_me` (`scripts/env_catalog.py:53`, `scripts/env-lib.sh:183`). A dry-run fill from `.env.example` reports the JWT placeholder as `would write`. Fix: import the catalogue's definition.

**STRC-04 (Medium)** `.env.example:183`: `#INSIDEOUT_AI_TURN_DEADLINE=12m`. The per-machine key manifests are a hand-written second catalogue, copied per machine, missing two skeleton variables and carrying one the skeleton lacks; the 31 per-key suites skip in CI. Fix: one shared manifest set and a name-equality test. Tracking of `hosts/` itself was decided as rv4-PUB-10 and is not refiled.

**STRC-05 (Medium)** `client/lib/features/project/roadmap_node_menu.dart:114`: `Future<void> _edit(BuildContext context, Appearance l10n) async {`. A 161-line method fetches, renders and writes, in a file at 349 lines. `roadmap_page.dart:38` holds loading, two stream loops and a 135-line build. Fix: `roadmap_controller.dart` and `roadmap_node_edit_dialog.dart`.

**STRC-06 (Medium)** `server/internal/linebudget/line_budget_test.go:27`: `const budget = 400`. The rule says 350, the client enforces 350, the server 400, scripts nothing. Seven client files sit at 330 to 350. `ci.yml` at 350 blocks two CI steps (open records R1-01, R3-20). Fix: one number, a script guard, the specified `ci.yml` split.

**STRC-07 (Medium)** `scripts/tests/check_docs_truth_rv4.py:6`: `Companion to check_docs_truth.py, split out to keep both under the line`. Five gates and 31 test files are named after a review round; `read` is defined 11 times and `compose_service` three. Fix: rename by subject, add `scripts/tests/_gate.py`.

**STRC-08 (Medium)** `client/lib/api/client_project.dart:27`: `Future<Map<String, dynamic>> getProject(String id) async {`. Roadmap verbs share a file with project verbs, and nine verbs return untyped maps. Fix: `client_roadmap.dart` and typed returns.

**STRC-09 (Low)** `scripts/`: five kinds of file at one level, tests in two places, a second folder at `server/scripts`. Fix: `scripts/tests/env/`, `scripts/checks/`, a README.

**STRC-10 (Low)** `client/test`: 156 flat files; the component library sits under `theme/`. Fix: mirror `lib/` and move the `block_*` widgets.

**STRC-11 (Low)** `scripts/bootstrap.sh:164`: `on_path() {`. A 26-line block identical to `scripts/install.sh:45-70` with no gate holding them equal.

**STRC-12 (Low)** `.gitignore:142`: `.grok/` is untracked and not ignored; skills live at the root and under `skills/`.

## Lead verification

| Lead | Verdict | The line that decided it |
| --- | --- | --- |
| 4 | Confirmed | The owner picker is `roadmap_node_menu.dart:152-161`; no subtree view exists in `features/project`. |
| 7 | Not re-run | I did not run the Go suites. |
| 8 | Confirmed, understated | Three gates are red, not one: exit codes 1, 2 and 1. |
| 9 | Confirmed for my area | Largest client files are 349 lines; `ci.yml` is 350; the Go guard is set to 400. |
| 12 | Confirmed, count moved | `git status --short` now shows 35 modified and 24 untracked; two are this review's plan files. |
| 14 | Checked | R1-01, R3-20, R3-38 and rv4-PUB-10 are cited, not refiled; none of STRC-01 to STRC-12 is on record. |

## Recommendations

1. Green the three gates before the working tree is committed.
2. Keep one installer: shim `scripts/setup.sh` onto the Go wizard.
3. One definition of placeholder, env parsing and env writing for the Python and shell tools.
4. One shared key manifest set with a CI-runnable equality test.
5. Extract the roadmap controller before building any per-owner tree view.
6. One budget number everywhere; treat a file parked just under it as a split signal.
7. Rename gates by subject and share their helpers.

## What this role would do first

I would fix the three red gates. It is about fifteen lines in three files, it is the difference between the uncommitted reset-watch and installer work being pushable or not, and every later structural change in this report is checked by those same gates. The next step is the `setup.sh` shim, because it removes a wrong behaviour on a documented path and deletes about two hundred lines instead of adding any.

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| STRC-01 | DOWNGRADED to Medium | The facts hold: of the 19 gates, 16 exit 0 and three fail in the working tree. Only one of the three is red on the pushed tip, and that one (check_docs_truth_rv4.py) is IDX-02's root cause. The two gates this finding adds, check_first_run_copy.py and check_supply_chain.py, pass on a clean clone of both 4d186da and c2900fd, so they fail only against the uncommitted scripts/install.sh and server/go.mod edits. That is unfinished in-flight work that bites when the slice is committed, not a red gate on the pushed tip. Not a restatement of an open docs/issues record. | `for f in scripts/tests/check_*.py; do python3 -I "$f"; done` in the working tree -> exit 1 check_docs_truth_rv4.py, exit 2 check_first_run_copy.py ("no api_base= closing block in scripts/install.sh"), exit 1 check_supply_chain.py ("NOT_LINKED lists github.com/erikgeiser/coninput ... drop it"), 16 others exit 0. Same loop in a /tmp clone at 4d186da and at c2900fd -> only check_docs_truth_rv4.py fails. |
| STRC-02 | DOWNGRADED to Medium | The unquoted write is real and reproduced: setup.sh reports it wrote DATABASE_URL, and the repo's own env check then reports it not set. The script is on a documented path (README.md:105, the install.sh prompt at :91 which is also in HEAD, SKILL.md:35, skills/insideout-setup/references/modules.md:148). I downgrade because the failure needs an own-database URL with a shell metacharacter (a single-parameter URL through the same script passes the check, and the default bundled choice writes a single-parameter URL), it fails closed before any migration, and docs/usage/installation.md names `insideout setup --local` as the guided path with setup.sh as the pre-binary fallback; the Go wizard handles the same URL correctly. If the coordinator weighs the install.sh prompt as the main funnel, High is arguable, since managed-Postgres URLs with two parameters are common. The duplication half is a recorded decision: docs/issues/2026-10-07-env-catalog-go-python-duplication.md (resolved 2026-10-10) keeps both the Go and Python readers behind a parity test; that test does not cover setup.sh's inline writer, so the quoting defect is new and not a restatement of an open record. The role-gate and non-atomic-write sub-claims were read, not probed. | `printf '<url>?sslmode=require&channel_binding=require' \| INSIDEOUT_ENV_FILE=/tmp/insideout-rv5/gates-and-index/env ./scripts/setup.sh --local --postgres own --no-start` -> "==> Wrote DATABASE_URL (not printed)", exit 0; then `INSIDEOUT_ENV_FILE=... ./scripts/env.sh check server` -> "FAIL: DATABASE_URL — not set in /tmp/insideout-rv5/gates-and-index/env"; bash sourcing gives DATABASE_URL=[UNSET] channel_binding=[require]. Controls: one-parameter URL through setup.sh -> "ok: DATABASE_URL", 0 failures; the two-parameter URL through `/tmp/insideout-rv5/insideout setup --local --postgres own --no-start` -> line written single-quoted, "ok: DATABASE_URL", 0 failures. |

## Ordered recommendations

1. Make the working tree's three red gates green before it is committed: update the anchor in scripts/tests/check_first_run_copy.py, drop two entries in scripts/tests/check_supply_chain.py, and add the two missing file names to docs/architecture/backend.md.
2. Keep one local-server installer. Turn scripts/setup.sh into a shim over `insideout setup`, delete its private .env writer, and point scripts/install.sh and README.md at the Go command.
3. Give the Python and shell env tools one definition each of placeholder, env-file parsing and env-file writing, owned by scripts/env_catalog.py, and have scripts/keys.py import it.
4. Replace the per-machine copies of the key manifests with one shared set and add a CI-runnable test that the manifest variable names equal the names in .env.example.
5. Extract a roadmap controller and a node edit dialog from the two roadmap files before building any subtree or per-owner view, so the new view reuses loading and live updates instead of copying them.
6. Use one line budget number for Go, Dart, shell, Python and workflow files, and treat a file parked within twenty lines of it as a prompt to split by responsibility, starting with the ci.yml split that is already specified.
7. Rename the review-round gates by subject and give them one shared helper module; do the same for test files as they are next touched, without a bulk rename.
8. Move the roadmap verbs of the Dart client into their own part file and return typed models from getProject and listNodeEvidence.
