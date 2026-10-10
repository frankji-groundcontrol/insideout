# Provider-shape legs (review-3 TEST-05), sourced by smoke.sh after the
# five surfaces: one real mail send and one real LLM coach turn, each only
# when its key is configured and each printing a SKIP line naming the
# missing variable otherwise. Uses the parent's call/expect_* helpers,
# PASS/FAIL counters, BASE, JAR and EMAIL1; the LLM leg stops the parent's
# offline server first (single-replica contract) and hands BASE to the
# real-key one.

echo
echo "=== P1: mail provider shape (one real probe send) ==="
# The auth email flows answer 200 regardless of delivery, so the only
# response-truthful mail surface is the admin probe (review-3 ARCH-03). It
# needs an admin: this run's own user is promoted through the owner pool —
# the sanctioned path, there is no self-promotion API — and left that way;
# every user here is a throwaway with a per-run name. The recipient must be
# the Resend account's verified address (the default sender delivers to
# nothing else), which is why SMOKE_MAIL_TO is configured, never invented.
if [ -z "${INSIDEOUT_MAIL_API_KEY:-}" ]; then
  echo "  SKIP: mail provider shape — INSIDEOUT_MAIL_API_KEY not set; mail runs in dev mode (logged, not delivered)"
elif [ -z "${SMOKE_MAIL_TO:-}" ]; then
  echo "  SKIP: mail provider shape — SMOKE_MAIL_TO not set; the default sender delivers only to the account's verified address"
elif [ -n "${SMOKE_BASE:-}" ] || [ ! -x "$TMP/insideout" ]; then
  echo "  SKIP: mail provider shape — needs the server booted by this script (promote-admin uses its owner pool)"
elif [ -z "${DATABASE_OWNER_URL:-}" ]; then
  echo "  SKIP: mail provider shape — no DATABASE_OWNER_URL to grant the probe's admin role"
else
  if "$TMP/insideout" promote-admin "$EMAIL1" >/dev/null 2>&1; then
    pass "promote-admin granted the probe's admin role"
    call POST /api/v1/admin/mail/probe "{\"email\":\"$SMOKE_MAIL_TO\"}" "$JAR"
    expect_code 200 "admin mail probe accepted 200"
    expect_field .status sent "provider accepted the send (status=sent)"
    [ "$(jget .status)" = "sent" ] || echo "    provider says: $(jget .error)"
  else
    fail "promote-admin for the mail probe (owner pool reachable?)"
  fi
fi

echo
echo "=== P2: LLM provider shape (one real coach turn) ==="
# S1/S2 above ran on the offline-template coach to stay deterministic, so
# they prove nothing about the provider. This leg (review-3 TEST-05) is the
# one place smoke talks to the real LLM: a second server with the key takes
# over the database — the single-replica contract (api/single_replica.go's
# advisory lock) allows exactly one api server per database, so the offline
# one is stopped first instead of run beside — and a single coach turn
# through it fails here, with logs, when the provider stops accepting our
# request shape, instead of on deploy day.
if [ -n "${SMOKE_BASE:-}" ]; then
  echo "  SKIP: LLM provider shape — SMOKE_BASE run; S1 above already exercised that server's coach"
elif [ -z "${INSIDEOUT_LLM_API_KEY:-}" ]; then
  echo "  SKIP: LLM provider shape — INSIDEOUT_LLM_API_KEY not set; only the offline template coach is covered (S1/S2)"
else
  PORT2=$(( (RANDOM % 20000) + 20000 )); BASE2="http://127.0.0.1:$PORT2"
  kill "$SRV_PID" 2>/dev/null || true
  wait "$SRV_PID" 2>/dev/null || true # exit ends the lock's session; sleep out the race
  SRV_PID=""; sleep 1
  echo "  real-key server taking over the database on $BASE2"
  INSIDEOUT_ADDR=":$PORT2" "$TMP/insideout" >>"$SERVER_LOG" 2>&1 &
  SRV2_PID=$!
  HEALTHY=""
  for _ in $(seq 1 60); do
    if curl -fsS "$BASE2/healthz" >/dev/null 2>&1; then HEALTHY=1; break; fi
    sleep 0.5
  done
  if [ -z "$HEALTHY" ]; then
    fail "real-key server healthy at $BASE2"
  else
    BASE="$BASE2" # the helpers below target the real-key server
    JAR3="$TMP/u3.jar"; : > "$JAR3"
    call POST /api/v1/auth/register "{\"email\":\"smoke+$RUN-c@example.com\",\"password\":\"smokepass123\",\"username\":\"smoke3$RUN\"}" "$JAR3"
    expect_code 201 "register user3 on the real-key server 201"
    call POST /api/v1/workspaces "{\"title\":\"WS2 $RUN\",\"description\":\"smoke\"}" "$JAR3"
    expect_code 201 "create workspace on the real-key server 201"; WS2="$(jget .id)"
    call POST "/api/v1/workspaces/$WS2/ideas" "{\"title\":\"Idea2 $RUN\",\"content\":\"a note-taking app for teams\"}" "$JAR3"
    expect_code 201 "create idea on the real-key server 201"; IDEA2="$(jget .id)"
    call POST "/api/v1/ideas/$IDEA2/convert" "" "$JAR3"
    expect_code 201 "convert idea on the real-key server 201"; CONV2="$(jget .conversationId)"
    CODE=""; : > "$SSEOUT"
    curl -sN --max-time 60 -b "$JAR3" -H 'Content-Type: application/json' \
      -d '{"content":"你好，帮我打磨这个想法"}' \
      "$BASE/api/v1/conversations/$CONV2/messages" >"$SSEOUT" 2>/dev/null || true
    BODY="$(head -c 240 "$SSEOUT")"
    grep -q '^event: message_start' "$SSEOUT" && pass "real-key SSE message_start" || fail "real-key SSE missing message_start"
    grep -q '^event: delta' "$SSEOUT" && pass "real-key SSE delta frame" || fail "real-key SSE missing delta"
    grep -q '^event: message_end' "$SSEOUT" && pass "real-key SSE message_end (the provider accepted the turn)" || fail "real-key SSE missing message_end"
    BASE="http://127.0.0.1:$PORT"
  fi
fi
