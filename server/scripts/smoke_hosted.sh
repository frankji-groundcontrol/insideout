# Hosted-mode probes of a running deployment (TEST-03), sourced by
# smoke.sh after its helpers: the five-surface suite stays in the parent,
# this fragment owns the read-only deployment checks. Uses the parent's
# call/expect_* helpers, PASS/FAIL counters, BASE and NOJAR; exits the
# parent shell when done.
# ---- hosted mode: read-only probes of a running deployment (TEST-03) -----
# The five surfaces below register users and write data, so they need an
# open-signup database of their own; the hosted deployment is
# invitation-only and must not be written to by a probe. Hosted mode
# instead proves the deployed stack answers at all: the SPA entry page
# (where a missing client image or a broken nginx upstream shows first,
# BUG-027), healthz, the public auth config, and the unauthenticated 401
# of BUG-027's owner check. The one authenticated probe is gated on
# INSIDEOUT_TOKEN (the CLI's session env) and skips with a message when
# it is not configured — a missing secret never fails a hosted run.
if [ "${SMOKE_HOSTED:-0}" = "1" ]; then
  echo
  echo "=== H: hosted deployment, read-only probes at $BASE ==="
  CODE="$(curl -sL -o "$BODYF" -w '%{http_code}' "$BASE/" || echo 000)"
  BODY="$(cat "$BODYF" 2>/dev/null || echo '')"
  expect_code 200 "hosted SPA entry page 200"
  if printf '%s' "$BODY" | grep -iq '<html'; then
    pass "entry page serves HTML (the client image is there)"
  else
    fail "entry page serves HTML (the client image is there)"
  fi
  call GET /healthz "" "$NOJAR"; expect_code 200 "hosted healthz 200"
  call GET /api/v1/auth/config "" "$NOJAR"; expect_code 200 "hosted auth config 200"
  expect_nonempty .signupMode "auth config carries signupMode"
  call GET /api/v1/me "" "$NOJAR"; expect_code 401 "hosted unauthenticated /me 401"
  if [ -n "${INSIDEOUT_TOKEN:-}" ]; then
    CODE="$(curl -s -o "$BODYF" -w '%{http_code}' \
      -H "Authorization: Bearer $INSIDEOUT_TOKEN" "$BASE/api/v1/me" || echo 000)"
    BODY="$(cat "$BODYF" 2>/dev/null || echo '')"
    expect_code 200 "hosted authenticated /me 200 (INSIDEOUT_TOKEN)"
  else
    echo "  SKIP: authenticated /me probe — INSIDEOUT_TOKEN not set (read-only probes only)"
  fi
  printf '=== RESULT: %d passed, %d failed ===\n' "$PASS" "$FAIL"
  [ "$FAIL" -eq 0 ] || exit 1
  echo "HOSTED PROBES GREEN"
  exit 0
fi
