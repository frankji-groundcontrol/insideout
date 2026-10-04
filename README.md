# InsideOut

> **跟踪进展、打磨想法、写出更好的 PRD**
> **Track the work. Refine the ideas. Ship better PRDs.**
>
> This README describes the current runnable build. The confirmed target
> product experience is in [`PRODUCT.md`](PRODUCT.md).

---

## 这是什么？ / What is this?

InsideOut 帮你的团队跟踪其他人正在开发的项目、随手记录想法的火花，并通过 AI 教练的引导式对话把想法打磨成一份扎实的 PRD。特别适合协作型团队或工作坊场景——组长能一眼看到所有项目的最新动态，每个成员都能零门槛记录自己的想法。

InsideOut helps your team keep track of projects other people are building, capture ideas the moment they strike, and turn them into a solid PRD through a guided AI coaching conversation. It's built for collaborative teams and workshop settings — a group leader gets an at-a-glance view of every project's latest update, and every member gets a frictionless way to record ideas.

## 核心功能 / Core Features

> 本节描述**当前可运行版本**的功能；确认中的目标产品体验见 [`PRODUCT.md`](PRODUCT.md)——其中已取代「僵化四阶段教练」与「管理员审批所有 PRD」的旧假设。
> This section describes the **current runnable build**; the confirmed target product experience lives in [`PRODUCT.md`](PRODUCT.md), which supersedes the rigid four-stage coach and the admin-approves-all-PRDs model.

### 📋 项目看板 / Project Board
工作区内所有项目共享一个看板，实时显示进度、阻塞和备注。滞后太久没更新的项目会被特别标出——不用追着问也知道发生了什么。
Every project in your workspace shows up on one shared board with live progress, blockers, and notes. Projects that have gone quiet too long get flagged — so you know what's happening without having to ask.

### 💡 想法收集箱 / Idea Inbox
每个成员都有一个零门槛的想法收集箱：一句话记下念头，等准备好了再打磨，或者直接转化成 PRD 开始细化。
Every member gets a frictionless idea inbox: jot down a thought in one sentence, come back to refine it whenever you're ready, or convert it straight into a PRD.

### 🤖 PRD 教练 / PRD Coach
把一个想法转化后，AI 教练会陪你走完四个阶段：**澄清**你的问题和目标用户 → **起草**完整的 PRD 章节 → **批判**指出薄弱环节 → **定稿**给出完整度总结。教练直接把内容写进 PRD 章节，你能实时看到它成形。
Convert an idea and an AI coach walks you through four stages: **clarify** the problem and target users → **draft** the full PRD sections → **critique** to call out weak spots → **finalize** with a completeness summary. The coach writes directly into your PRD sections — you watch it take shape live.

### ✅ 评审流程 / Review Flow
PRD 写好后提交评审，工作区管理员通过或驳回；被驳回的 PRD 可以修改后重新提交。每次快照都会保存为一个版本，随时可以回看历史。
Submit a finished PRD for review; a workspace admin approves or rejects it. A rejected PRD can be revised and resubmitted. Every snapshot is saved as a version you can revisit anytime.

### 📤 导出 / Export
随时把 PRD 导出为 Markdown 下载，或用浏览器打印功能另存为 PDF。
Export any PRD to Markdown for download, or use your browser's print function to save it as a PDF.

## 快速上手 / Quick Start

1. 注册账号（邮箱 + 密码）/ Register an account (email + password)
2. 创建一个工作区，或用邀请码加入已有的工作区 / Create a workspace, or join an existing one with an invite code
3. 在项目看板里创建你的第一个项目 / Create your first project on the board
4. 去想法收集箱记下一个念头 / Jot an idea down in the inbox
5. 点「转化为 PRD」，开始和教练对话 / Hit "Convert to PRD" and start talking to the coach

> 注册默认是「仅限邀请」：新数据库播种 `signup_mode = invitation_only`，没有邀请码注册会返回 403。本地首次启动请先用 `insideout promote-admin <your-email>` 提升管理员、再生成邀请码（仅限本地开发可改用 `insideout signup-mode open`）。详见 [docs/usage/cli-mcp-and-guidance.md](docs/usage/cli-mcp-and-guidance.md)。
> Registration is **invitation-only by default**: a fresh database seeds `signup_mode = invitation_only` and registering without an invite code returns 403. For a local first run, bootstrap with `insideout promote-admin <your-email>` and mint invites (or `insideout signup-mode open` for local dev only). See [docs/usage/cli-mcp-and-guidance.md](docs/usage/cli-mcp-and-guidance.md).

## 本地开发 / Local Development

全部命令在仓库根目录执行 / all commands from the repo root:

```bash
# 配置环境 / Set up the environment
./scripts/env.sh init            # 交互式，绝不打印值 / interactive, never prints values

# 后端 / Backend
./scripts/dev.sh -C server go run ./cmd/insideout migrate   # 应用数据库迁移 / apply migrations
./scripts/dev.sh -C server go run ./cmd/insideout seed      # 可选：演示数据 / optional: demo data
./scripts/dev.sh -C server go run ./cmd/insideout           # 启动服务 / start the server

# 前端 / Frontend (production is Flutter web)
cd client && flutter run -d chrome
```

托管站点 / hosted: `https://<your-app>.up.railway.app`（你的 Railway 部署地址 / your own Railway deployment's URL）

`dev.sh` 是必需的：应用本身不读取 `.env`（Go 只读进程环境变量），必须先由 `dev.sh` 导出。它每次启动前都会执行 `env.sh check`，缺项时明确报错。
`dev.sh` is not optional: nothing in the app reads `.env` itself (Go reads the process environment), so it must be exported first — a bare `cd server && go run` sees none of your values. `dev.sh` also runs `env.sh check` before every launch and names what is missing.

只有两个变量是必填的：`DATABASE_URL`（任意 PostgreSQL 14+ 实例）和 `INSIDEOUT_JWT_SECRET`（至少 32 字符）。未设置 `INSIDEOUT_LLM_API_KEY` 时教练使用离线模板回复，适合本地开发。
Only two variables are required: `DATABASE_URL` (any PostgreSQL 14+ instance) and `INSIDEOUT_JWT_SECRET` (min 32 chars) — they are the only two left uncommented in `.env.example`. Without `INSIDEOUT_LLM_API_KEY` the coach falls back to an offline template reply, handy for local dev. Step-by-step: [docs/SETENV.md](docs/SETENV.md); every variable: [docs/usage/environment.md](docs/usage/environment.md).

或者直接用 docker-compose 一键启动后端技术栈（postgres + server）；Flutter 前端需另行启动（`cd client && flutter run -d chrome`）：
Or bring up the backend stack (postgres + server) with docker-compose — the Flutter frontend runs separately (`cd client && flutter run -d chrome`):

```bash
docker compose up -d
```

## 了解更多 / Learn More

- 产品定义 / Product definition: [`PRODUCT.md`](PRODUCT.md)
- 开发计划 / Development plan: [`docs/PLAN.md`](docs/PLAN.md)
- 完整技术方案 / Full technical plan: [`docs/plans/2026-07-20-go-rewrite/`](docs/plans/2026-07-20-go-rewrite/README.md)
- 开发进度 / Progress tracker: [`docs/TODO.md`](docs/TODO.md)

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
