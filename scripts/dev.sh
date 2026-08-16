#!/usr/bin/env bash
# Run a dev command with the root .env exported into the environment.
# 导出根目录 .env 后在指定子模块中执行开发命令。
#
# Roles / 分工: env.sh sets .env up and validates it (init/check); dev.sh runs
# things. dev.sh runs `env.sh check` first and refuses to launch when a
# required key is missing — the check output names exactly what to fix.
# env.sh 负责 .env 的配置与校验（init/check）；dev.sh 负责运行。dev.sh 先执行
# `env.sh check`，缺少必需项时拒绝启动——校验输出会指明缺什么、如何补。
#
# Nothing in the app loads .env itself — the Go server reads plain process
# env (os.Getenv) and Nuxt reads NUXT_* env — so .env must be exported before
# the process starts. -C selects which directory to run in. docker-compose is
# the other bridge: it interpolates ${VAR} from .env on its own.
# 应用本身都不读取 .env——Go 服务读进程环境变量（os.Getenv），Nuxt 读 NUXT_*
# 环境变量——因此 .env 必须在进程启动前导出。-C 指定在哪个子模块中运行；
# docker-compose 是另一座桥，会自行对 .env 做 ${VAR} 插值。
#
# Usage / 用法 (from the repo root / 在仓库根目录执行):
#   ./scripts/dev.sh -C server go run ./cmd/insideout
#   ./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v
#   ./scripts/dev.sh -C app pnpm dev
#
# It never prints env values; the env.sh preflight fails loudly on missing or
# invalid required keys (run ./scripts/env.sh init to set them).
# 绝不打印变量值；必需项缺失或非法时由 env.sh 预检明确报错（用 ./scripts/env.sh init 补齐）。
set -euo pipefail

if [ "${1:-}" != "-C" ] || [ "$#" -lt 3 ]; then
  echo "usage: ./scripts/dev.sh -C <server|app> <command> [args...]" >&2
  echo "  e.g. ./scripts/dev.sh -C server go run ./cmd/insideout" >&2
  exit 2
fi

dir="$2"
shift 2

case "$dir" in
  server|app) ;;
  *) echo "dev.sh: unknown component '$dir' — expected 'server' or 'app'" >&2; exit 2 ;;
esac

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/.." && pwd)"

if [ ! -f "$here/env.sh" ]; then
  echo "dev.sh: $here/env.sh is missing — the repo's scripts/ directory is incomplete" >&2
  exit 1
fi

# Preflight: validate required keys (never prints values). On failure env.sh
# tells the user exactly what is missing and exits non-zero, so we stop here.
# Passing the component makes the gate specific: a generated component .env
# that was built from an OLDER root file is a FAILURE for that component (it
# would silently supply stale values to `cd <dir> && pnpm dev`) and only a
# warning for the others, so a stale app/.env cannot block a server test run.
# 预检：校验必需项（绝不打印值）。传入组件名后，该组件的陈旧副本会导致失败，
# 其它组件仅告警——避免陈旧的 app/.env 阻塞 server 端测试。
"$here/env.sh" check "$dir" || exit 1

set -a
. "$repo/.env"
set +a

cd "$repo/$dir"
exec "$@"
