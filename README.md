# InsideOut

> **跟踪进展、打磨想法、写出更好的 PRD**
> **Track the work. Refine the ideas. Ship better PRDs.**

---

## 这是什么？ / What is this?

InsideOut 帮你的团队跟踪其他人正在开发的项目、随手记录想法的火花，并通过 AI 教练的引导式对话把想法打磨成一份扎实的 PRD。特别适合协作型团队或工作坊场景——组长能一眼看到所有项目的最新动态，每个成员都能零门槛记录自己的想法。

InsideOut helps your team keep track of projects other people are building, capture ideas the moment they strike, and turn them into a solid PRD through a guided AI coaching conversation. It's built for collaborative teams and workshop settings — a group leader gets an at-a-glance view of every project's latest update, and every member gets a frictionless way to record ideas.

## 核心功能 / Core Features

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

## 本地开发 / Local Development

```bash
# 后端 / Backend
cd server
go run ./cmd/insideout migrate   # 应用数据库迁移 / apply migrations
go run ./cmd/insideout seed      # 可选：创建演示数据 / optional: create demo data
go run ./cmd/insideout           # 启动服务 / start the server

# 前端 / Frontend
cd app
pnpm install
pnpm dev
```

需要在 `.env` 中设置 `DATABASE_URL`（任意 PostgreSQL 14+ 实例）和 `INSIDEOUT_JWT_SECRET`；未设置 `ANTHROPIC_AUTH_TOKEN` 时教练会用离线模板回复，方便本地开发。完整变量说明见 `.env.example`。
You need `DATABASE_URL` (any PostgreSQL 14+ instance) and `INSIDEOUT_JWT_SECRET` set in `.env`; without `ANTHROPIC_AUTH_TOKEN` the coach falls back to an offline template reply, handy for local dev. See `.env.example` for every variable.

或者直接用 docker-compose 一键启动完整技术栈：
Or bring up the whole stack with docker-compose:

```bash
docker compose up -d
```

## 了解更多 / Learn More

- 产品定义 / Product definition: [`docs/INIT.md`](docs/INIT.md)
- 开发计划 / Development plan: [`docs/PLAN.md`](docs/PLAN.md)
- 完整技术方案 / Full technical plan: [`docs/plans/2026-07-20-go-rewrite/`](docs/plans/2026-07-20-go-rewrite/README.md)
- 开发进度 / Progress tracker: [`docs/TODO.md`](docs/TODO.md)

## 第三方资源 / Third-party assets

**字体 / Fonts.** 本项目的界面字体是 **阿里巴巴普惠体 2.0（Alibaba PuHuiTi 2.0）**，设计上是全局无衬线体（见 `app/tailwind.config.js` 的 `fontFamily.sans`）。

- 版权 / Copyright: © 2020-2021 阿里巴巴（中国）有限公司，版权所有 / Alibaba (China) Co., Ltd. All rights reserved.
- 商标 / Trademark: 阿里巴巴是阿里巴巴集团控股有限公司的商标 / Alibaba is a trademark of Alibaba Group Holding Limited.
- 制作 / Manufacturer: Alibaba Design；汉仪字库 / Hanyi Fonts. 版本 2.00，字符集 GB18030-2000。
- 官方发布 / Official source: <https://alibabafont.taobao.com/>（阿里巴巴字体官方门户 / the Alibaba Fonts portal).

**许可说明 / License note.** 阿里巴巴普惠体由阿里巴巴官方免费发布，允许个人与商业**使用**；但它**不是**开源许可证（非 OSI/OFL），字体文件标注「版权所有（All rights reserved）」，嵌入权限为 Preview & Print（fsType=4）。因此「免费使用」不等于「可再分发字体文件」——把字体二进制提交进本仓库属于再分发，超出了授权范围。**本仓库不包含、不分发任何字体文件**：我们只在字体栈中按名称**引用**它。已安装该字体的访问者会看到它，其他访问者回退到 Noto Sans SC（经 Google Fonts）及系统 CJK 字体。展示衬线体 Noto Serif SC 同样通过 Google Fonts 引用。如需在你的环境还原完全一致的观感，请自行从上方官方渠道获取并安装（这属于你自己的「使用」）。

**Body typeface.** The app's UI sans is **Alibaba PuHuiTi 2.0** (see `fontFamily.sans` in `app/tailwind.config.js`). It is published by Alibaba for free personal and commercial **use**, but it is **not** an open-source (OSI/OFL) license — the font file is marked "All rights reserved" with Preview & Print embedding (fsType=4). "Free to use" is not "free to redistribute the font file," so committing the binary to this repo would exceed the grant. **This repository contains and distributes no font files.** We reference the font by name only: visitors who have it installed render it; everyone else falls back to Noto Sans SC (via Google Fonts) and the platform CJK face. The display serif, Noto Serif SC, is likewise referenced via Google Fonts. To reproduce the exact look in your own environment, obtain and install the font yourself from the official source above (that is your own "use").
