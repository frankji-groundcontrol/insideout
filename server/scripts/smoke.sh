#!/usr/bin/env bash
# Live functional smoke test for all five product surfaces, driven over real
# HTTP against a real PostgreSQL (no mocks). It boots the server on a random
# high port, registers fresh uniquely-named users, then exercises:
#   S1 PRD coach SSE streaming
#   S2 AI roadmap (build-from-PRD, expand, CRUD)
#   S3 GitHub repo link + commit sync
#   S4 project-updates timeline (progress/blocker/note add/edit/delete)
#   S5 cross-tenant authz + unauthenticated negatives
#   P1 mail provider shape — one real send through the admin mail probe,
#     only when INSIDEOUT_MAIL_API_KEY and SMOKE_MAIL_TO (the
#     Resend-verified sink address) are set
#   P2 LLM provider shape — one real coach turn, only when
#     INSIDEOUT_LLM_API_KEY is set (the five surfaces above run on the
#     offline template and stay deterministic; the real-key server takes
#     the database over afterwards, one replica at a time)
# Both provider legs print a SKIP line naming the missing variable when they
# cannot run — a skipped leg is never silent (review-3 TEST-05).
# Rerun-safe: every entity is named with a per-run suffix, so re-runs never
# collide with prior rows. Exits non-zero on the first failed assertion count.
#
# Needs the server's env (DATABASE_URL, INSIDEOUT_JWT_SECRET; a repo-root .env
# is loaded when there is one, CI exports them itself) and signup mode "open",
# because it registers its own users. Exit 2 means that precondition is unmet.
# It also reaches GitHub's public API once (S3; a 429/404/502 there passes)
# and, when configured, the LLM and mail providers once (P1/P2).
#
# Usage:
#   cd server && ./scripts/smoke.sh            # boots its own server
#   SMOKE_BASE=http://127.0.0.1:54321 ./scripts/smoke.sh   # use a running one
#   SMOKE_SERVER_LOG=/path/server.log ./scripts/smoke.sh   # keep the boot log
#   SMOKE_HOSTED=1 SMOKE_BASE=https://host ./scripts/smoke.sh   # read-only
#     probes of a running deployment (TEST-03): the SPA entry page, healthz,
#     the public auth config and an unauthenticated 401, plus an
#     authenticated /me when INSIDEOUT_TOKEN is set. It registers no users
#     and writes nothing, so it needs neither a database nor open signup.
#
# ponytail: one bash script + curl + jq is the smallest live check that fails
# loudly if any surface breaks; upgrade to a Go httptest harness only if this
# grows beyond ~5 surfaces or needs parallel tenants.
set -euo pipefail

cd "$(dirname "$0")/.." # run from server/
REPO_ROOT="$(cd .. && pwd)"

# ---- load env without ever printing it (sanctioned pattern) -----------------
# A checkout without a .env (CI) supplies the variables itself.
if [ -f "$REPO_ROOT/.env" ]; then set -a; source "$REPO_ROOT/.env"; set +a; fi

PORT="${SMOKE_PORT:-$(( (RANDOM % 20000) + 40000 ))}"
BASE="${SMOKE_BASE:-http://127.0.0.1:$PORT}"
RUN="$RANDOM$(date +%s)"
TMP="$(mktemp -d)"
JAR="$TMP/u1.jar"; JAR2="$TMP/u2.jar"; NOJAR="$TMP/none.jar"
SSEOUT="$TMP/sse.txt"; BODYF="$TMP/body"
SERVER_LOG="${SMOKE_SERVER_LOG:-$TMP/server.log}"
: > "$NOJAR"

PASS=0; FAIL=0
SRV_PID=""; SRV2_PID=""

cleanup() {
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null || true
  [ -n "$SRV2_PID" ] && kill "$SRV2_PID" 2>/dev/null || true
  rm -rf "$TMP"
}
trap cleanup EXIT

pass() { PASS=$((PASS+1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); printf '  \033[31mFAIL\033[0m %s\n    code=%s body=%.240s\n' "$1" "${CODE:-?}" "${BODY:-}"; }

# Hosted mode only ever probes, so it must not fall through to booting a
# server of its own: without SMOKE_BASE it has nothing to probe.
if [ "${SMOKE_HOSTED:-0}" = "1" ] && [ -z "${SMOKE_BASE:-}" ]; then
  echo "hosted mode probes a running deployment: set SMOKE_BASE to its public URL" >&2
  exit 2
fi

# ---- boot the server unless SMOKE_BASE points at one already ----------------
if [ -z "${SMOKE_BASE:-}" ]; then
  echo "building + booting server on $BASE (run $RUN)"
  # This server keeps the offline template coach/planner so the SSE + roadmap
  # surfaces stay deterministic — the key is cleared per-command below, not
  # exported, so the provider leg's own boot (P1) still inherits the real
  # one. A configured mail key makes config demand a public URL (emailed
  # links would otherwise point at localhost); no smoke mail carries a link
  # anyone follows, so default it to this server's own address.
  if [ -n "${INSIDEOUT_MAIL_API_KEY:-}" ] && [ -z "${INSIDEOUT_PUBLIC_URL:-}" ]; then
    export INSIDEOUT_PUBLIC_URL="$BASE"
  fi
  export INSIDEOUT_ADDR=":$PORT" INSIDEOUT_COOKIE_SECURE=0
  go build -o "$TMP/insideout" ./cmd/insideout
  INSIDEOUT_LLM_API_KEY= "$TMP/insideout" >"$SERVER_LOG" 2>&1 &
  SRV_PID=$!
  for _ in $(seq 1 60); do
    if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then break; fi
    sleep 0.5
  done
fi
curl -fsS "$BASE/healthz" >/dev/null || { echo "server not healthy at $BASE"; [ -f "$SERVER_LOG" ] && tail -20 "$SERVER_LOG"; exit 1; }

# ---- helpers ----------------------------------------------------------------
# call METHOD PATH [JSONDATA] [JARFILE]  -> sets CODE and BODY globals
call() {
  local method=$1 path=$2 data=${3:-} jar=${4:-$JAR}
  local args=(-s -o "$BODYF" -w '%{http_code}' -X "$method" "$BASE$path")
  [ -n "$jar" ] && args+=(-b "$jar" -c "$jar")
  [ -n "$data" ] && args+=(-H 'Content-Type: application/json' -d "$data")
  CODE="$(curl "${args[@]}" || echo 000)"
  BODY="$(cat "$BODYF" 2>/dev/null || echo '')"
}
jget() { printf '%s' "$BODY" | jq -r "$1 // empty" 2>/dev/null || echo ''; }

expect_code() { [ "$CODE" = "$1" ] && pass "$2" || fail "$2 (want $1)"; }
expect_any() { # expect_any "403 404" LABEL
  for c in $1; do [ "$CODE" = "$c" ] && { pass "$2"; return; }; done
  fail "$2 (want one of: $1)"
}
expect_field() { # expect_field JQPATH WANT LABEL
  local got; got="$(jget "$1")"
  [ "$got" = "$2" ] && pass "$3" || fail "$3 (field $1 = '$got', want '$2')"
}
expect_nonempty() { # expect_field_nonempty JQPATH LABEL
  local got; got="$(jget "$1")"
  [ -n "$got" ] && pass "$2" || fail "$2 (field $1 empty)"
}

source "$REPO_ROOT/server/scripts/smoke_hosted.sh"


# A fresh database is invitation-only and answers 403 to a bare register, so
# say so up front instead of failing every later step.
call GET /api/v1/auth/config "" "$NOJAR"
MODE="$(jget .signupMode)"
if [ "$MODE" != "open" ]; then
  echo "smoke registers its own users, so signup must be open; this server reports signupMode='${MODE:-unknown}' (HTTP ${CODE})." >&2
  echo "Open it first (insideout signup-mode open) and put it back afterwards." >&2
  exit 2
fi

echo
echo "=== S0: health + auth ==="
call GET /healthz "" "$NOJAR"; expect_code 200 "healthz 200"

EMAIL1="smoke+$RUN-a@example.com"; EMAIL2="smoke+$RUN-b@example.com"
call POST /api/v1/auth/register "{\"email\":\"$EMAIL1\",\"password\":\"smokepass123\",\"username\":\"smoke$RUN\"}" "$JAR"
expect_code 201 "register user1 201"
U1="$(jget .id)"; [ -n "$U1" ] && pass "register returned user id" || fail "register missing id"
call GET /api/v1/me "" "$JAR"; expect_code 200 "me via session cookie 200"
call GET /api/v1/me "" "$NOJAR"; expect_code 401 "me unauthenticated 401"

echo
echo "=== setup: workspace -> project, idea -> prd+conversation ==="
call POST /api/v1/workspaces "{\"title\":\"WS $RUN\",\"description\":\"smoke\"}" "$JAR"
expect_code 201 "create workspace 201"; WS="$(jget .id)"
call POST "/api/v1/workspaces/$WS/projects" "{\"title\":\"Proj $RUN\",\"description\":\"smoke\"}" "$JAR"
expect_code 201 "create project 201"; PROJ="$(jget .id)"
call POST "/api/v1/workspaces/$WS/ideas" "{\"title\":\"Idea $RUN\",\"content\":\"a note-taking app for teams\"}" "$JAR"
expect_code 201 "create idea 201"; IDEA="$(jget .id)"
call POST "/api/v1/ideas/$IDEA/convert" "" "$JAR"
expect_code 201 "convert idea 201"; PRD="$(jget .prdId)"; CONV="$(jget .conversationId)"
[ -n "$PRD" ] && [ -n "$CONV" ] && pass "convert returned prdId+conversationId" || fail "convert missing prdId/conversationId"

echo
echo "=== S1: PRD coach SSE streaming ==="
CODE=""; : > "$SSEOUT"
curl -sN --max-time 30 -b "$JAR" -H 'Content-Type: application/json' \
  -d '{"content":"你好，帮我打磨这个想法"}' \
  "$BASE/api/v1/conversations/$CONV/messages" >"$SSEOUT" 2>/dev/null || true
BODY="$(head -c 240 "$SSEOUT")"
grep -q '^event: message_start' "$SSEOUT" && pass "SSE message_start" || fail "SSE missing message_start"
grep -q '^event: delta' "$SSEOUT" && pass "SSE delta frame" || fail "SSE missing delta"
grep -q '^event: message_end' "$SSEOUT" && pass "SSE message_end (stream completed)" || fail "SSE missing message_end"
call GET "/api/v1/conversations/$CONV/messages" "" "$JAR"
expect_code 200 "list conversation messages 200"

echo
echo "=== S2: AI roadmap (draft / confirm / expand / CRUD) ==="
# Build and expand are two steps: the first answers 200 with a draft and writes
# nothing; the second (.../confirm) writes what the caller sends back.
call POST "/api/v1/prds/$PRD/build" "" "$JAR"
expect_code 200 "draft roadmap from PRD 200"; BPROJ="$(jget .projectId)"
TREE="$(printf '%s' "$BODY" | jq -c '.tree // empty' 2>/dev/null || true)"
[ -n "$BPROJ" ] && [ -n "$TREE" ] && pass "draft carries a project and a tree" || fail "draft missing projectId or tree"
call POST "/api/v1/prds/$PRD/build/confirm" "{\"tree\":${TREE:-null}}" "$JAR"
expect_code 201 "confirm drafted roadmap 201"; NNODES="$(jget .nodeCount)"
[ -n "$NNODES" ] && [ "$NNODES" != "0" ] && pass "build produced $NNODES nodes" || fail "confirm nodeCount empty/zero"
call GET "/api/v1/projects/$BPROJ/roadmap" "" "$JAR"
expect_code 200 "list roadmap 200"
NODE="$(jget '.[0].id')"
[ -n "$NODE" ] && pass "roadmap has a node to expand" || fail "no roadmap node found"
call POST "/api/v1/roadmap/$NODE/expand" "" "$JAR"
expect_code 200 "draft subtasks for the node 200"
CHILDREN="$(printf '%s' "$BODY" | jq -c '.children // empty' 2>/dev/null || true)"
[ -n "$CHILDREN" ] && [ "$CHILDREN" != "[]" ] && pass "expand drafted subtasks" || fail "expand drafted no children"
call POST "/api/v1/roadmap/$NODE/expand/confirm" "{\"children\":${CHILDREN:-[]}}" "$JAR"
expect_code 201 "confirm drafted subtasks 201"
call POST "/api/v1/projects/$BPROJ/roadmap" "{\"title\":\"Manual node $RUN\"}" "$JAR"
expect_code 201 "create roadmap node 201"; MNODE="$(jget .id)"
call PATCH "/api/v1/roadmap/$MNODE" "{\"status\":\"in_progress\"}" "$JAR"
expect_code 200 "patch roadmap node status 200"; expect_field .status in_progress "patched status persisted"
call POST "/api/v1/roadmap/$MNODE/move" "{\"position\":1}" "$JAR"; expect_code 200 "move roadmap node 200"
call DELETE "/api/v1/roadmap/$MNODE" "" "$JAR"; expect_code 200 "delete roadmap node 200"

echo
echo "=== S3: GitHub repo link + commit sync ==="
call PUT "/api/v1/projects/$PROJ/repo" '{"repoUrl":"not a url"}' "$JAR"
expect_code 400 "set invalid repoUrl rejected 400"
call PUT "/api/v1/projects/$PROJ/repo" '{"repoUrl":"https://github.com/octocat/Hello-World"}' "$JAR"
expect_code 200 "set valid repoUrl 200"; expect_field .repoUrl "https://github.com/octocat/Hello-World" "repoUrl persisted"
# sync without a linked repo -> 400 (use a fresh repo-less project)
call POST "/api/v1/workspaces/$WS/projects" "{\"title\":\"Proj2 $RUN\"}" "$JAR"; PROJ2="$(jget .id)"
call POST "/api/v1/projects/$PROJ2/sync-github" "" "$JAR"
expect_code 400 "sync with no repo linked 400"
# real sync: 200 (commits added) or 429 (rate-limited) or 404/502 (upstream) all
# prove endpoint+authz+validation; only client/server errors are real failures.
call POST "/api/v1/projects/$PROJ/sync-github" "" "$JAR"
case "$CODE" in
  200|429) pass "sync linked repo ($CODE)";;
  404|502) pass "sync linked repo ($CODE, upstream-dependent — not a product defect)";;
  *) fail "sync linked repo (want 200/429/404/502)";;
esac
# The sync response's `added` count is the source of truth for commit review;
# commits land in the repo timeline (not project_updates), so we don't assert
# latestUpdateKind here.
if [ "$CODE" = 200 ]; then
  ADDED="$(jget .added)"
  [ -n "$ADDED" ] && pass "sync returned 200, added=$ADDED commits" || fail "sync 200 but no added count"
fi

echo
echo "=== S4: project-updates timeline ==="
call POST "/api/v1/projects/$PROJ/updates" '{"kind":"milestone"}' "$JAR"
expect_code 400 "invalid update kind rejected 400"
call POST "/api/v1/projects/$PROJ/updates" "{\"kind\":\"progress\",\"content\":\"progress $RUN\"}" "$JAR"
expect_code 201 "add progress update 201"; UPD="$(jget .id)"; expect_field .kind progress "kind persisted"
call POST "/api/v1/projects/$PROJ/updates" "{\"kind\":\"blocker\",\"content\":\"blocker $RUN\"}" "$JAR"
expect_code 201 "add blocker update 201"
call POST "/api/v1/projects/$PROJ/updates" "{\"kind\":\"note\",\"content\":\"note $RUN\"}" "$JAR"
expect_code 201 "add note update 201"
call PATCH "/api/v1/updates/$UPD" "{\"content\":\"edited $RUN\"}" "$JAR"
expect_code 200 "edit update 200"; expect_field .content "edited $RUN" "edited content persisted"
call GET "/api/v1/projects/$PROJ" "" "$JAR"
expect_code 200 "get project (embeds updates) 200"; expect_nonempty '.updates[0].id' "updates embedded in project view"
call DELETE "/api/v1/updates/$UPD" "" "$JAR"; expect_code 200 "delete update 200"

echo
echo "=== S5: cross-tenant authz + negatives ==="
call POST /api/v1/auth/register "{\"email\":\"$EMAIL2\",\"password\":\"smokepass123\",\"username\":\"smoke2$RUN\"}" "$JAR2"
expect_code 201 "register user2 201"
call GET "/api/v1/workspaces/$WS" "" "$JAR2"; expect_any "403 404" "user2 cannot read foreign workspace"
call GET "/api/v1/projects/$PROJ" "" "$JAR2"; expect_any "403 404" "user2 cannot read foreign project"
call POST "/api/v1/projects/$PROJ/updates" '{"kind":"note","content":"x"}' "$JAR2"; expect_any "403 404" "user2 cannot post to foreign project"
call GET "/api/v1/conversations/$CONV/messages" "" "$JAR2"; expect_any "403 404" "user2 cannot read foreign conversation"
call POST "/api/v1/prds/$PRD/build" "" "$JAR2"; expect_any "403 404" "user2 cannot build from foreign prd"
call PATCH "/api/v1/updates/$UPD" '{"content":"hijack"}' "$JAR2"; expect_any "403 404" "user2 cannot edit foreign update"
call GET "/api/v1/projects/$PROJ" "" "$NOJAR"; expect_code 401 "unauthenticated project read 401"
source "$REPO_ROOT/server/scripts/smoke_providers.sh"


echo
printf '=== RESULT: %d passed, %d failed ===\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ] || { [ -f "$SERVER_LOG" ] && { echo "--- server log tail ---"; tail -30 "$SERVER_LOG"; }; exit 1; }
echo "ALL SURFACES GREEN"
