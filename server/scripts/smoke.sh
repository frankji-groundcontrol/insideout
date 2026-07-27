#!/usr/bin/env bash
# Live functional smoke test for all five product surfaces, driven over real
# HTTP against a real PostgreSQL (no mocks). It boots the server on a random
# high port, registers fresh uniquely-named users, then exercises:
#   S1 PRD coach SSE streaming
#   S2 AI roadmap (build-from-PRD, expand, CRUD)
#   S3 GitHub repo link + commit sync
#   S4 project-updates timeline (progress/blocker/note add/edit/delete)
#   S5 cross-tenant authz + unauthenticated negatives
# Rerun-safe: every entity is named with a per-run suffix, so re-runs never
# collide with prior rows. Exits non-zero on the first failed assertion count.
#
# Usage:
#   cd server && ./scripts/smoke.sh            # boots its own server
#   SMOKE_BASE=http://127.0.0.1:54321 ./scripts/smoke.sh   # use a running one
#
# ponytail: one bash script + curl + jq is the smallest live check that fails
# loudly if any surface breaks; upgrade to a Go httptest harness only if this
# grows beyond ~5 surfaces or needs parallel tenants.
set -euo pipefail

cd "$(dirname "$0")/.." # run from server/
REPO_ROOT="$(cd .. && pwd)"

# ---- load env without ever printing it (sanctioned pattern) -----------------
set -a; source "$REPO_ROOT/.env"; set +a

PORT="${SMOKE_PORT:-$(( (RANDOM % 20000) + 40000 ))}"
BASE="${SMOKE_BASE:-http://127.0.0.1:$PORT}"
RUN="$RANDOM$(date +%s)"
TMP="$(mktemp -d)"
JAR="$TMP/u1.jar"; JAR2="$TMP/u2.jar"; NOJAR="$TMP/none.jar"
SSEOUT="$TMP/sse.txt"; BODYF="$TMP/body"
: > "$NOJAR"

PASS=0; FAIL=0
SRV_PID=""

cleanup() {
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null || true
  rm -rf "$TMP"
}
trap cleanup EXIT

pass() { PASS=$((PASS+1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); printf '  \033[31mFAIL\033[0m %s\n    code=%s body=%.240s\n' "$1" "${CODE:-?}" "${BODY:-}"; }

# ---- boot the server unless SMOKE_BASE points at one already ----------------
if [ -z "${SMOKE_BASE:-}" ]; then
  echo "building + booting server on $BASE (run $RUN)"
  # Force the offline template coach/planner so the SSE + roadmap surfaces are
  # deterministic (no live LLM needed). Pass SMOKE_BASE to test a real-key server.
  export INSIDEOUT_ADDR=":$PORT" INSIDEOUT_COOKIE_SECURE=0 ANTHROPIC_AUTH_TOKEN=
  go build -o "$TMP/insideout" ./cmd/insideout
  "$TMP/insideout" >"$TMP/server.log" 2>&1 &
  SRV_PID=$!
  for _ in $(seq 1 60); do
    if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then break; fi
    sleep 0.5
  done
fi
curl -fsS "$BASE/healthz" >/dev/null || { echo "server not healthy at $BASE"; [ -f "$TMP/server.log" ] && tail -20 "$TMP/server.log"; exit 1; }

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
echo "=== S2: AI roadmap (build / expand / CRUD) ==="
call POST "/api/v1/prds/$PRD/build" "" "$JAR"
expect_code 201 "build roadmap from PRD 201"; BPROJ="$(jget .projectId)"; NNODES="$(jget .nodeCount)"
[ -n "$NNODES" ] && [ "$NNODES" != "0" ] && pass "build produced $NNODES nodes" || fail "build nodeCount empty/zero"
call GET "/api/v1/projects/$BPROJ/roadmap" "" "$JAR"
expect_code 200 "list roadmap 200"
NODE="$(jget '.[0].id')"
[ -n "$NODE" ] && pass "roadmap has a node to expand" || fail "no roadmap node found"
call POST "/api/v1/roadmap/$NODE/expand" "" "$JAR"
expect_code 201 "expand roadmap node 201"
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

echo
printf '=== RESULT: %d passed, %d failed ===\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ] || { [ -f "$TMP/server.log" ] && { echo "--- server log tail ---"; tail -30 "$TMP/server.log"; }; exit 1; }
echo "ALL SURFACES GREEN"
