# InsideOut

> **跟踪进展、打磨想法、写出更好的 PRD**
> **Track the work. Refine the ideas. Ship better PRDs.**
>
> This README describes the current runnable build. The confirmed target
> product experience is in [`PRODUCT.md`](PRODUCT.md).

---

## 这是什么？ / What is this?

InsideOut 把一个想法一路带到可交付：**想法收集箱**随手记录念头 → 转化为 **PRD**（八个章节，可按受众导出）→ 长成 **路线图**（带截止日期的项目树）→ 由**人类提交（Commit）**定格为版本。绑定 GitHub 仓库后，提交、PR 和部署会作为**证据**自动挂到对应的路线图节点上——进展有据可查，而不是靠嘴说。协作是实时的：画布上能看到谁在线、光标在哪。

InsideOut walks an idea all the way to shipped: capture sparks in the
**idea inbox** → convert them into a **PRD** (eight sections, exportable per
audience) → grow a **roadmap** (a project tree with deadlines) → pin it as a
version with a human-only **Commit**. Bind a GitHub repository and commits,
PRs, and deployments attach themselves to roadmap nodes as **evidence** —
progress you can verify, not claims you have to trust. Collaboration is
live: presence and cursors on the canvas, and a time-first progress view.

三个使用面，同一套 `/api/v1` 契约：

Three surfaces, one `/api/v1` contract:

- **Web 应用** — Flutter 网页端：想法箱、PRD 编辑、AI 教练对话、路线图画布、版本历史。
- **`insideout` CLI** — 终端里的全部动词：认证、想法箱、PRD、路线图、版本、证据、代理词汇。
- **`insideout-mcp` MCP 服务器** — 同样的动词作为 MCP 工具暴露给任何支持 MCP 的编码代理（当前数量以 `tools/list` 为准，不在文档里写死）；让 Claude Code、Cursor 或你自己的代理直接驱动这个产品。

- **Web app** — the Flutter client: inbox, PRD editing, the AI coach
  conversation, the roadmap canvas, version history.
- **`insideout` CLI** — every verb in the terminal: auth, idea inbox,
  PRDs, roadmap, versions, evidence, the agent vocabulary.
- **`insideout-mcp` MCP server** — the same verbs as MCP tools for any
  MCP-capable coding agent (`tools/list` answers the current count, so no
  number is frozen here); let Claude Code, Cursor, or your own agent
  drive the product directly.

## 安装 / Install

### 网页端 / Web

打开你的部署地址即可（注册默认**仅限邀请**，见下方快速上手）。自托管见 [docs/usage/deployment.md](docs/usage/deployment.md)。

Open your deployment's URL (signup is **invitation-only** by default — see
Quick start below). For self-hosting, see
[docs/usage/deployment.md](docs/usage/deployment.md).

### CLI 与 MCP（从源码构建 / from source）

在一份仓库检出里执行（需要 Go 1.26+）：

From a checkout of this repository (needs Go 1.26+):

```bash
./scripts/install.sh
```

默认安装到 `~/.local/bin`（可用第一个参数或 `INSIDEOUT_INSTALL_DIR` 覆盖），并在该目录不在 PATH 时自动把它写进 shell 的 rc 文件（zsh → `~/.zshrc`，bash → `~/.bashrc`，其余 → `~/.profile`）。脚本可重复执行。装好后 CLI 与 MCP 直接调用托管 API，不必设置环境变量；`./scripts/default-api.sh` 打印该地址。自建服务（含 PostgreSQL）用 `insideout setup`，没有二进制时才使用 `./scripts/setup.sh`，完整步骤见 [docs/usage/installation.md](docs/usage/installation.md)。没有脚本时的手动路径：`cd server && go build -o /usr/local/bin/insideout ./cmd/insideout`（`insideout-mcp` 同理）。

Installs both binaries into `~/.local/bin` by default (override with the
first argument or `INSIDEOUT_INSTALL_DIR`) and, when that directory is not
on PATH, appends one marked export line to your shell rc (zsh →
`~/.zshrc`, bash → `~/.bashrc`, otherwise `~/.profile`). Idempotent.
The CLI and MCP server then call the hosted API with no environment
variables. `./scripts/default-api.sh` prints that URL. Sign in with
`insideout login <email>`. A server on this machine, including
PostgreSQL, is `./scripts/setup.sh`. The full walk-through is
[docs/usage/installation.md](docs/usage/installation.md).
Manual fallback without the script: `cd server && go build -o
/usr/local/bin/insideout ./cmd/insideout` (same for `insideout-mcp`).

装好后，一条命令就是安装向导：交互式表单选择远程（托管/自建，密码、邮箱验证码或
magic link 登录，密钥隐藏输入）或本地（PostgreSQL 引导配置、迁移、健康探测），
结果存入 0600 配置文件；代理用标志驱动同一入口：

Once installed, one command is the wizard: an interactive form picks
remote (hosted/self-hosted; password, emailed code, or magic-link
sign-in, secrets hidden) or local (guided PostgreSQL, migrations,
health probes), saving the result to a 0600 config file — agents
drive the same door with flags:

```bash
insideout setup                       # 交互式 / interactive
insideout setup --local --postgres bundled   # 本地服务 / own server
insideout setup --check               # 健康探测 / re-run health probes
```

从零开始的新机器（macOS / Ubuntu）？让你的编码代理加载安装技能，按模块装齐
工具链、配置环境并留下安装记录：

Starting from a bare machine (macOS / Ubuntu)? Have your coding agent
load the installation skill — module-by-module toolchain install,
environment setup, and a written installation record:
[`skills/insideout-setup/SKILL.md`](skills/insideout-setup/SKILL.md).

把 MCP 服务器注册进你的代理（同一段 JSON 适用于 Claude Code、Cursor 等标准 `mcpServers` 配置）：

Register the MCP server with your agent (the same JSON snippet ports
across Claude Code, Cursor, and anything speaking the standard
`mcpServers` shape):

```bash
./scripts/mcp-config.sh                  # 打印说明 + JSON / hints + JSON
./scripts/mcp-config.sh -q >> some.json  # 只输出 JSON / JSON only
```

### CLI 与 MCP（预编译二进制 / prebuilt binaries）

`v0.1.0`（2026-10-11）发布了四个目标平台的 tarball、校验和与安装
脚本；仓库已于同日转为公开，下面这条命令已验证可用（bootstrap、
tarball 与校验和均以匿名请求 200 通过）。

`v0.1.0` (2026-10-11) publishes tarballs, checksums, and the installer
for all four targets; the repository went public the same day, and the
one-liner below is verified working (bootstrap, tarball, and checksum
all answered 200 anonymously).

```bash
curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash
```

### 本地开发 / Local development

全部命令在仓库根目录执行 / all commands from the repo root:

```bash
./scripts/env.sh init                    # 交互式生成 .env，绝不打印值
./scripts/dev.sh -C server go run ./cmd/insideout migrate   # 应用迁移
./scripts/dev.sh -C server go run ./cmd/insideout           # 启动服务
cd client && flutter run -d chrome --web-port=5173          # 前端 / frontend
docker compose up -d                     # 或一键后端栈 / or the backend stack
```

前端固定在 5173 端口：服务端按 `INSIDEOUT_PUBLIC_URL`（默认 `http://localhost:5173`）生成邀请、magic link 与重置链接，而 Flutter 默认随机选端口，不固定的话这些链接会指向无人监听的端口。

Port 5173 is pinned because the server builds invite, magic-link and reset
links from `INSIDEOUT_PUBLIC_URL` (default `http://localhost:5173`); Flutter
otherwise picks a random port, and those links would point at a port
nothing listens on.

`dev.sh` 是必需的：应用只读进程环境变量，由它先导出 `.env` 并做缺项检查。必填只有 `DATABASE_URL`（任意 PostgreSQL 14+）和 `INSIDEOUT_JWT_SECRET`（≥32 字符）；不设 `INSIDEOUT_LLM_API_KEY` 时教练走离线模板。详见 [docs/SETENV.md](docs/SETENV.md) 与 [docs/usage/local-development.md](docs/usage/local-development.md)。

`dev.sh` is not optional: nothing reads `.env` itself (Go reads the
process environment), so it exports it first and checks for missing keys.
Only two variables are required — `DATABASE_URL` (any PostgreSQL 14+) and
`INSIDEOUT_JWT_SECRET` (min 32 chars); without `INSIDEOUT_LLM_API_KEY` the
coach falls back to an offline template. Step by step:
[docs/SETENV.md](docs/SETENV.md) and
[docs/usage/local-development.md](docs/usage/local-development.md).

## 基本用法 / Basic usage

网页端的产品环：注册（或凭邀请码）→ 建工作区 → 在想法箱记一句话 → 「转化为 PRD」→ 和教练对话打磨 → 长出路线图、挂上仓库 → 人类 Commit 定格版本。

The web loop: register (invite code in hand) → create a workspace → jot a
one-line idea → "Convert to PRD" → refine it with the coach → grow the
roadmap and bind a repo → a human Commits the version.

CLI 一瞥（令牌只进环境变量，不进对话记录）：

A CLI taste (the token lives in your environment, never in a transcript):

```bash
insideout whoami                       # hosted API; no INSIDEOUT_API required
insideout login you@example.com        # password on stdin, token on stdout
insideout workspaces                   # 工作区列表 / list workspaces
insideout ideas <ws-id>                # 想法箱 / the inbox
insideout prd update --section goals="Ship the MVP" <prd-id>
insideout roadmap list <proj-id>       # 路线图 / the roadmap tree
insideout commit --name "v0.1" --audience decision <prd-id>   # 人类专属 / humans only
```

代理词汇——代理汇报、人类决策：`agent-context`（简报）、`checkpoint`（记录工作）、`propose`（提议变更）、`proposal-decide`（人类批准或驳回）。完整动词表见 [docs/usage/cli-mcp-and-guidance.md](docs/usage/cli-mcp-and-guidance.md)。

The agent vocabulary — agents report, humans decide: `agent-context`
(the briefing), `checkpoint` (record work), `propose` (propose changes),
`proposal-decide` (a human accepts or rejects). Full verb table:
[docs/usage/cli-mcp-and-guidance.md](docs/usage/cli-mcp-and-guidance.md).

> 注册默认「仅限邀请」：全新数据库播种 `signup_mode = invitation_only`，无邀请码注册返回 403。本地首跑先用 `insideout promote-admin <your-email>` 提升管理员、再生成邀请码（本地开发可 `insideout signup-mode open`）。
> Registration is **invitation-only by default**: a fresh database seeds
> `signup_mode = invitation_only`; registering without an invite code
> returns 403. For a local first run, bootstrap with `insideout
> promote-admin <your-email>` then mint invites (`insideout signup-mode
> open` for local dev only).

## 了解更多 / Learn more

| 想了解 / Want to know | 去看 / Read |
| --- | --- |
| 产品定义 / Product definition | [`PRODUCT.md`](PRODUCT.md) |
| 设计准则 / Design law | [`DESIGN.md`](DESIGN.md) |
| 文档总地图 / Doc map | [`docs/index.md`](docs/index.md) |
| CLI / MCP / GitHub 证据环 / CLI, MCP, evidence loop | [`docs/usage/cli-mcp-and-guidance.md`](docs/usage/cli-mcp-and-guidance.md) |
| 安装 / Installation | [`docs/usage/installation.md`](docs/usage/installation.md) |
| 密钥设置教程 / Key setup tutorial | [`docs/usage/key-setup-tutorial.md`](docs/usage/key-setup-tutorial.md) |
| 密钥测试教程 / Key testing tutorial | [`docs/usage/key-testing-tutorial.md`](docs/usage/key-testing-tutorial.md) |
| 本地开发 / Local development | [`docs/usage/local-development.md`](docs/usage/local-development.md) |
| 环境变量 / Environment variables | [`docs/usage/environment.md`](docs/usage/environment.md) · [`docs/SETENV.md`](docs/SETENV.md) |
| 部署与备份 / Deploy + backup | [`docs/usage/deployment.md`](docs/usage/deployment.md) · [`docs/usage/backup-restore.md`](docs/usage/backup-restore.md) |
| 架构 / Architecture | [`docs/architecture/index.md`](docs/architecture/index.md) |
| 变更记录 / Change log | [`docs/changelogs/`](docs/changelogs/README.md) |
| 贡献与安全 / Contributing & security | [`CONTRIBUTING.md`](CONTRIBUTING.md) · [`SECURITY.md`](SECURITY.md) |
| 许可证 / License | [`LICENSE`](LICENSE) - Mozilla Public License 2.0 |

## 第三方资源 / Third-party assets

**字体 / Fonts.**

- 原生构建（iOS / Android / 桌面端）附带两份 SIL OFL 许可的可变字体：**Noto Sans SC** 与
  **Noto Serif SC**（`client/assets/fonts/`，经 `client/pubspec.yaml` 打包，由
  `client/lib/theme/native_fonts.dart` 加载）。许可全文随附于
  [`client/assets/fonts/OFL.txt`](client/assets/fonts/OFL.txt)。
- Web 端不打包字体文件，经 Google Fonts 加载所需字族（Archivo、Noto Sans SC、
  Source Serif 4、Noto Serif SC；见 `client/lib/theme/web_fonts_web.dart`）。
- Native builds (iOS / Android / desktop) ship two SIL OFL-licensed variable
  fonts, **Noto Sans SC** and **Noto Serif SC** (`client/assets/fonts/`,
  bundled via `client/pubspec.yaml`, loaded by
  `client/lib/theme/native_fonts.dart`). The full license text ships alongside
  as [`client/assets/fonts/OFL.txt`](client/assets/fonts/OFL.txt).
- The web build loads its faces from Google Fonts instead of bundling font
  files (Archivo, Noto Sans SC, Source Serif 4, Noto Serif SC; see
  `client/lib/theme/web_fonts_web.dart`).

**阿里巴巴普惠体 / Alibaba PuHuiTi — 历史署名，仅作记录 / historical attribution only:**

- 版权 / Copyright: © 2020-2021 阿里巴巴（中国）有限公司，版权所有 / Alibaba (China) Co., Ltd. All rights reserved.
- 商标 / Trademark: 阿里巴巴是阿里巴巴集团控股有限公司的商标 / Alibaba is a trademark of Alibaba Group Holding Limited.
- 制作 / Manufacturer: Alibaba Design；汉仪字库 / Hanyi Fonts. 版本 2.00，字符集 GB18030-2000。
- 官方发布 / Official source: <https://alibabafont.taobao.com/>（阿里巴巴字体官方门户 / the Alibaba Fonts portal).

**许可说明 / License note.** 阿里巴巴普惠体由阿里巴巴官方免费发布，允许个人与商业**使用**；但它**不是**开源许可证（非 OSI/OFL），字体文件标注「版权所有（All rights reserved）」，嵌入权限为 Preview & Print（fsType=4）。因此「免费使用」不等于「可再分发字体文件」——把字体二进制提交进本仓库属于再分发，超出了授权范围。**本仓库不包含、不分发普惠体**：它曾是已删除的 Nuxt Ink & Seal UI 字体栈中「按名称引用」的成员，当前的 Cobalt 客户端并不引用它；上方署名仅为历史记录保留。如需在你的环境还原旧观感，请自行从上方官方渠道获取并安装（这属于你自己的「使用」）。
Alibaba PuHuiTi is released free for personal and commercial **use**, but it is **not** an open-source license (neither OSI nor OFL): the files are marked "All rights reserved" with Preview & Print embedding permission (fsType=4). Free to use does not mean free to redistribute — committing the binaries to this repo would be redistribution beyond the grant. **This repository does not contain or distribute PuHuiTi**: it was referenced by name in the deleted Nuxt Ink & Seal UI's font stack, and the current Cobalt client does not reference it. The attribution above is kept as the historical record.
