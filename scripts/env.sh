#!/usr/bin/env bash
# env.sh — owns the ENVIRONMENT: inspect, validate, set up, propagate.
# env.sh——负责环境：查看、校验、配置、传播。
#
# Roles / 分工: env.sh owns the environment, scripts/dev.sh owns the processes,
# and dev.sh gates every launch on `env.sh check <component>`. Neither does the
# other's job; the checker is the launcher's door.
# env.sh 管环境，dev.sh 管进程；dev.sh 每次启动前都以 `env.sh check` 为门禁。
#
# Usage / 用法 (repo root / 仓库根目录):
#   ./scripts/env.sh init                    # interactive setup / 交互式配置
#   ./scripts/env.sh edit [--list]           # every variable + its state; set/clear
#   ./scripts/env.sh list                    # key NAMES only / 仅键名
#   ./scripts/env.sh redact                  # KEY=<redacted> — safe to paste anywhere
#   ./scripts/env.sh check [server|client] [--db]   # validate; exit 1 on failure
#   ./scripts/env.sh propagate [component]   # no-op unless a component owns a .env.example
#
# A SECRET's value is never printed: list gives names, redact gives
# KEY=<redacted>, check gives a status (secrets as 'set (N chars)', DSNs with
# the userinfo blanked, unreducible DSN shapes masked whole). What counts as a
# secret is decided by the variable NAME, never by inspecting its value.
# Non-secret flags ARE echoed on purpose — "only exactly '1' enables it" is
# only actionable when it quotes what you wrote.
# 绝不打印密钥值；是否为密钥按变量名判断。非密钥的开关值会回显，以便定位问题。
#
# INSIDEOUT_ENV_FILE=<path> validates a different file (testing) / 测试时可指定其他文件
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/.." && pwd)"
ENV_FILE="${INSIDEOUT_ENV_FILE:-$repo/.env}"
EXAMPLE_FILE="$repo/.env.example"

. "$here/env-lib.sh"

# ── check · non-interactive validation, dev.sh's preflight gate · 非交互校验 ──

# A suffix naming the shell-export-shadows-the-file situation, or nothing.
shadow_note() {
  case " $shadowed " in
    *" $1 "*) printf '%s' " — note: your shell exports $1, so THIS process sees a value, but docker compose reads the file" ;;
  esac
}

# exact_flag NAME EXPECT OK_DETAIL WARN_DETAIL — only exactly EXPECT flips it.
exact_flag() {
  local v="${!1:-}"
  [ -n "$v" ] || return 0
  if [ "$v" = "$2" ]; then ok "$1" "$3"; else warn "$1" "'$v' — $4"; fi
}

# Defaults in effect for unset optionals — awareness only, none blocks startup.
# 未设可选项的生效默认值——仅提示，均不阻塞启动。Row format: NAME|default|consumer.
report_defaults() {
  local spec name dflt who
  for spec in \
    "INSIDEOUT_ADDR|:8080|Go server listen address" \
    "INSIDEOUT_ACCESS_TTL|15m|Go server access-token lifetime" \
    "INSIDEOUT_REFRESH_TTL|720h|Go server refresh-token lifetime" \
    "INSIDEOUT_COOKIE_SECURE|on (Secure cookies)|Go server; exactly '0' for plain-http dev" \
    "INSIDEOUT_DEV_CORS|off|Go server; exactly '1' for permissive CORS" \
    "INSIDEOUT_CORS_ORIGINS|empty (no allow-list CORS)|Go server Flutter-web origins; token localhost = any localhost port" \
    "INSIDEOUT_LLM_BASE_URL|https://api.anthropic.com/v1|Go server LLM endpoint (include /v1 yourself)" \
    "INSIDEOUT_LLM_MODEL|claude-sonnet-4-20250514|Go server LLM model (ignored by the offline coach)" \
    "INSIDEOUT_LLM_SCHEMA|messages|Go server LLM wire format: messages or responses"; do
    name="${spec%%|*}"; spec="${spec#*|}"; dflt="${spec%%|*}"; who="${spec#*|}"
    [ -n "${!name:-}" ] && continue
    info "$name" "default in effect: $dflt — $who"
  done
  return 0  # never let the `&& continue` above be the loop's last command (set -e)
}

# Generated component copies: is each one still derived from THIS root file?
# Fail for the component being launched (so dev.sh stops), warn for the others.
# No component currently owns a .env.example (server and client are `-`).
# 生成副本是否仍源自当前根文件；陈旧副本对目标组件报 FAIL，对其它组件仅警告。
check_copies() {
  local focus="${1:-}" c dir out want have body_want body_have
  want="$(root_sum "$ENV_FILE")"
  for c in $COMPONENTS; do
    dir="$(component_dir "$c")"
    [ -n "$dir" ] && [ "$dir" != - ] || continue
    out="$repo/$dir/.env"
    if [ ! -f "$out" ]; then
      if [ "$c" = "$focus" ]; then
        info "$dir/.env" "not generated yet — ./scripts/env.sh propagate $c (the code defaults apply until then)"
      fi
      continue
    fi
    if ! grep -qF "$GEN_MARK" "$out"; then
      warn "$dir/.env" "hand-written, not generated — its keys win over the root if that component reads dotenv. Move it aside and run ./scripts/env.sh propagate $c"
      continue
    fi
    have="$(sed -n "s/^${SUM_MARK} //p" "$out" | head -1 || true)"
    body_have="$(sed -n "s/^${BODY_MARK} //p" "$out" | head -1 || true)"
    body_want="$(body_sum "$out")"
    if [ -n "$body_have" ] && [ "$body_have" != "$body_want" ]; then
      # Edited after generation. The source stamp cannot see this — it
      # fingerprints the root file, not the copy — so without the body stamp a
      # tampered copy reported "in sync" while feeding the component a value
      # the root never held, and the next propagate discarded the edit in
      # silence (the DO-NOT-EDIT marker was still there, so nothing refused).
      if [ "$c" = "$focus" ]; then
        fail "$dir/.env" "EDITED since it was generated — it no longer matches the root. Discard it: rm $dir/.env && ./scripts/env.sh propagate $c — or keep it deliberately by deleting its '# generated by' header, which makes it a hand-written override"
      else
        warn "$dir/.env" "edited since it was generated — ./scripts/env.sh propagate $c to discard, or delete its header to keep it as a deliberate override"
      fi
    elif [ "$have" = "$want" ]; then
      ok "$dir/.env" "generated, in sync with $ENV_FILE"
    elif [ "$c" = "$focus" ]; then
      fail "$dir/.env" "generated from an OLDER .env — re-run: ./scripts/env.sh propagate $c"
    else
      warn "$dir/.env" "generated from an OLDER .env — re-run: ./scripts/env.sh propagate $c"
    fi
  done
  return 0
}

do_check() {
  local probe_db=0 comp="" name val url_pw load_out probe_url
  n_ok=0; n_warn=0; n_fail=0
  while [ "$#" -gt 0 ]; do
    case "$1" in
      "")          : ;;
      --db)        probe_db=1 ;;
      server|client)  comp="$1" ;;
      *) echo "usage: ./scripts/env.sh check [server|client] [--db]" >&2; exit 2 ;;
    esac
    shift
  done
  if [ ! -f "$ENV_FILE" ]; then
    echo "FAIL: $ENV_FILE not found — run ./scripts/env.sh init (or cp .env.example .env)" >&2
    exit 1
  fi
  # Record which required names your SHELL exports, before load_env clears
  # them. The file is the authority — compose reads it,
  # not your shell — but "empty — required" is baffling when `echo $DATABASE_URL`
  # shows a value, so name the situation instead of leaving it to be guessed.
  # 以文件为准，但若 shell 中导出了同名变量则明确说明，避免困惑。
  shadowed=""
  for name in DATABASE_URL INSIDEOUT_JWT_SECRET; do
    if [ -n "${!name:-}" ]; then shadowed="$shadowed $name"; fi
  done
  load_out="$(load_env)"; eval "$load_out"
  info file "$ENV_FILE: this file is the authority; each line below names its consumer / 以此文件为准，消费方见下"
  # A placeholder in a REQUIRED variable is a FAILURE, not a warning. Both
  # skeleton values pass config.Load — the DSN is non-empty and the secret is
  # 33 characters — so a bare `cp .env.example .env` would otherwise sail
  # through this gate and fail later with an opaque DNS error. A check that
  # cannot fail in the direction of the bug is not a check. For OPTIONAL
  # variables a placeholder stays a warning: nothing breaks.
  # 必填项中的占位值视为失败而非警告——否则直接复制骨架即可通过校验，
  # 却在连接数据库时才报出难以理解的错误。可选项的占位值仍只是警告。
  # DATABASE_URL — required (server/internal/config fails fast) / 必填
  if [ -z "${DATABASE_URL:-}" ]; then
    fail DATABASE_URL "not set in $ENV_FILE — required (the Go server fails fast without it)$(shadow_note DATABASE_URL)"
  elif is_placeholder "$DATABASE_URL"; then
    fail DATABASE_URL "still the .env.example placeholder — set a real DSN (see SETENV.md step 1)"
  else
    ok DATABASE_URL "$(redact_url "$DATABASE_URL")"
  fi
  # INSIDEOUT_JWT_SECRET — required, min 32 chars / 必填，至少 32 字符
  if [ -z "${INSIDEOUT_JWT_SECRET:-}" ]; then
    fail INSIDEOUT_JWT_SECRET "not set in $ENV_FILE — required$(shadow_note INSIDEOUT_JWT_SECRET)"
  elif [ "${#INSIDEOUT_JWT_SECRET}" -lt 32 ]; then
    fail INSIDEOUT_JWT_SECRET "too short: ${#INSIDEOUT_JWT_SECRET} chars — need at least 32"
  elif is_placeholder "$INSIDEOUT_JWT_SECRET"; then
    fail INSIDEOUT_JWT_SECRET "still the .env.example placeholder — generate one: openssl rand -hex 32"
  else
    ok INSIDEOUT_JWT_SECRET "set (${#INSIDEOUT_JWT_SECRET} chars)"
  fi
  # TTLs — approximate Go time.ParseDuration (2h45m / 1.5h valid), when set / 仅设置时校验
  local dur_re='^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$'
  for name in INSIDEOUT_ACCESS_TTL INSIDEOUT_REFRESH_TTL; do
    val="${!name:-}"
    [ -n "$val" ] || continue
    if [[ "$val" =~ $dur_re ]]; then ok "$name" "$val"
    else warn "$name" "'$val' — does not approximate a Go duration (e.g. 15m, 720h)"; fi
  done
  # COOKIE_SECURE / DEV_CORS — only exactly '0' / '1' flip them / 仅 '0' / '1' 生效
  exact_flag INSIDEOUT_COOKIE_SECURE 0 "'0' — Secure cookie off (plain-http dev)" \
    "only exactly '0' disables Secure cookies; anything else keeps the secure default"
  exact_flag INSIDEOUT_DEV_CORS 1 "'1' — permissive CORS on" \
    "only exactly '1' enables permissive CORS; anything else leaves it off"
  # INSIDEOUT_LLM_API_KEY — empty means offline template-reply coach / 留空即离线模式
  if [ -z "${INSIDEOUT_LLM_API_KEY:-}" ]; then
    info INSIDEOUT_LLM_API_KEY "empty — offline template-reply coach (Go server)"
  else
    ok INSIDEOUT_LLM_API_KEY "$(mask_secret "$INSIDEOUT_LLM_API_KEY") (Go server)"
  fi
  # GITHUB_TOKEN — optional; empty keeps GitHub sync at the unauthenticated rate limit / 可选
  if [ -n "${GITHUB_TOKEN:-}" ]; then ok GITHUB_TOKEN "$(mask_secret "$GITHUB_TOKEN") (Go server)"
  else info GITHUB_TOKEN "empty — GitHub sync runs at the unauthenticated rate limit (Go server, optional)"; fi
  # POSTGRES_APP_PASSWORD — required ONLY for the bundled-postgres setup, which
  # is why .env.example leaves it commented: a flat 'required' would cry wolf
  # for everyone on a remote DSN. Conditional requirements belong in the
  # checker, not in the schema. / 仅自带 postgres 用法下必填，故骨架中保持注释。
  if is_bundled_pg; then
    if [ -z "${POSTGRES_APP_PASSWORD:-}" ]; then
      fail POSTGRES_APP_PASSWORD "empty — required by docker-compose for the bundled postgres"
    else
      url_pw="$(url_password "$DATABASE_URL")"
      if [ -n "$url_pw" ] && [ "$url_pw" != "$POSTGRES_APP_PASSWORD" ]; then
        warn POSTGRES_APP_PASSWORD "does not match the password embedded in DATABASE_URL"
      else ok POSTGRES_APP_PASSWORD "$(mask_secret "$POSTGRES_APP_PASSWORD")"; fi
    fi
  fi
  # --db: probe via pg_isready. Two transformations, both required:
  #   · strip the pgx-only pgbouncer param (libpq rejects it / libpq 不认该参数)
  #   · strip the userinfo. pg_isready only needs host and port, and argv is
  #     not private — /proc/<pid>/cmdline is world-readable by default, so
  #     passing the DSN whole would expose the database password to any local
  #     process for the duration of the probe. 参数表并不私密，故先去掉用户信息。
  # A libpq keyword/value DSN (`host=… password=…`) has no safe reduction, so
  # it is not probed rather than leaked.
  if [ "$probe_db" = 1 ]; then
    probe_url="$(printf '%s' "${DATABASE_URL:-}" | sed -E -e 's/\?pgbouncer=[^&]*&/?/' -e 's/&pgbouncer=[^&]*&/\&/' -e 's/[?&]pgbouncer=[^&]*$//' -e 's/\?$//' -e 's#(postgres(ql)?://)[^/]*@#\1#')"
    if [ -z "${DATABASE_URL:-}" ]; then info pg_isready "skipped — DATABASE_URL is empty"
    elif ! command -v pg_isready >/dev/null 2>&1; then info pg_isready "not installed — skipped"
    elif [ "${probe_url#postgres}" = "$probe_url" ]; then
      info pg_isready "skipped — a keyword/value DSN cannot be reduced to host+port without putting the password in the process table"
    elif pg_isready -d "$probe_url" -t 5 >/dev/null 2>&1; then ok pg_isready "database accepts connections"
    else fail pg_isready "database not reachable"; fi
  fi
  check_copies "$comp"
  report_defaults
  echo "$n_ok ok, $n_warn warning(s), $n_fail failure(s)${comp:+ — for $comp}"
  if [ "$n_fail" -gt 0 ]; then
    echo "hint: ./scripts/env.sh init (guided) or ./scripts/env.sh edit (pick a variable)"
    exit 1
  fi
  exit 0
}

# ── read-only views · 只读视图 ───────────────────────────────────────────────
do_list() {
  [ -f "$ENV_FILE" ] || { echo "FAIL: $ENV_FILE not found" >&2; exit 1; }
  sed -nE 's/^[[:space:]]*(export )?([A-Za-z_][A-Za-z0-9_]*)=.*/\2/p' "$ENV_FILE" | sort
}
do_redact() {
  local k
  while IFS= read -r k; do printf '%s=<redacted>\n' "$k"; done < <(do_list)
}

# ── usage · 用法 ─────────────────────────────────────────────────────────────
usage() {
  cat <<'EOF'
env.sh — set up, inspect, validate and propagate the environment. Values are
never printed. scripts/dev.sh preflights `check` before every launch.
env.sh——配置、查看、校验与传播环境变量；绝不打印值；dev.sh 每次启动前自动预检。

Usage (repo root / 仓库根目录):
  init                     interactive setup / 交互式配置
  edit [--list]            every variable + its state; enter sets, c clears
  list                     key names only / 仅键名
  redact                   KEY=<redacted> — safe to paste anywhere
  check [server|client] [--db]  validate; --db also probes the database
  propagate [component]    regenerate component .env copies (none today)

INSIDEOUT_ENV_FILE=<path> validates a different file (testing).
EOF
}

case "${1:-}" in
  init)      . "$here/env-write.sh"; do_init ;;
  propagate) . "$here/env-write.sh"; shift; do_propagate "$@" ;;
  check)     shift; do_check "$@" ;;
  list)      do_list ;;
  redact)    do_redact ;;
  # All catalog logic lives in scripts/env_catalog.py; `edit --list` renders
  # the same rows without a tty, which is what makes the TUI testable.
  edit)      shift; exec python3 "$here/env_tui.py" "$@" ;;
  -h|--help) usage ;;
  *)         usage >&2; exit 2 ;;
esac
