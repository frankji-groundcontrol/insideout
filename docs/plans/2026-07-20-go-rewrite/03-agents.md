# 03 — PRD Coach Agent / PRD 教练智能体

> Part of the [InsideOut rewrite plan](README.md). / [InsideOut 重写计划](README.md)的一部分。

## 1. Product Flow / 产品流程

A member records an idea in the workspace inbox. When ready, they hit **Convert** — the server creates a `prd` (empty sections) plus an `agent_conversation`, and the user lands in the PRD workspace: sections on the left, coach chat on the right. The coach guides them through four stages; when the PRD is solid, the author submits it for review and a workspace admin approves (通过) or rejects it. An approved PRD can be linked to a `project` so execution is tracked on the board.
成员在工作区收集箱记录想法。准备好后点击**转化**——服务端创建 `prd`（空章节）和 `agent_conversation`，用户进入 PRD 工作台：左侧章节、右侧教练对话。教练引导四个阶段；PRD 成熟后作者提交评审，工作区管理员通过或驳回。通过的 PRD 可关联 `project`，在看板上跟踪落地。

This reuses the app's proven interaction loop (write + AI sidebar + adopt), upgraded: the agent can now *write into the document itself* via tool calls, and history is server-side.
这复用了应用已验证的交互闭环（写作 + AI 侧栏 + 采纳），并升级：Agent 现在能通过工具调用*直接写入文档*，历史也改为服务端存储。

## 2. PRD Template / PRD 模板

`prds.sections` is a JSONB object of markdown strings, fixed keys / `prds.sections` 是固定键名的 markdown 字符串 JSONB 对象：

| Key | Section / 章节 |
|---|---|
| `background` | Background & problem statement / 背景与问题陈述 |
| `users` | Target users & personas / 目标用户与画像 |
| `goals` | Goals & success metrics / 目标与成功指标 |
| `nonGoals` | Non-goals / 非目标 |
| `stories` | User stories & scenarios / 用户故事与场景 |
| `requirements` | Functional requirements / 功能需求 |
| `constraints` | Constraints & dependencies / 约束与依赖 |
| `risks` | Risks & open questions / 风险与开放问题 |

Fixed keys make tool calls trivially validatable, render order deterministic, and the critique stage's completeness scoring mechanical (each section: empty / thin / solid).
固定键名让工具调用易校验、渲染顺序确定、批判阶段的完整度评分可机械化（每节：空 / 单薄 / 扎实）。

## 3. Agent Architecture / 智能体架构

**One agent, four stages** — a state machine on `agent_conversations.stage`, not a multi-agent zoo. Each stage swaps the system prompt; the model advances the stage itself via a tool call when its exit criteria are met (the user can also jump stages manually).
**一个 Agent、四个阶段**——`agent_conversations.stage` 上的状态机，而非多智能体动物园。每阶段切换系统提示词；退出条件满足时模型自行通过工具调用推进阶段（用户也可手动跳转）。

| Stage / 阶段 | Behavior / 行为 | Exit / 退出条件 |
|---|---|---|
| `clarify` 澄清 | Interview the user about the idea — **one focused question at a time** (problem, who hurts, current workaround, why now). Never drafts yet. / 就想法采访用户——**一次只问一个聚焦问题**（问题、痛在谁、现有替代、为何现在）。绝不提前起草。 | Problem, users, and value are stated concretely / 问题、用户、价值已具体明确 |
| `draft` 起草 | Writes every section via `update_prd_section` tool calls; user watches the PRD fill in live, then reacts. / 通过 `update_prd_section` 写满各章节；用户实时看到 PRD 成形并反馈。 | All 8 sections non-empty / 8 个章节皆非空 |
| `critique` 批判 | Scores each section empty/thin/solid, names the weakest two, asks targeted questions, patches sections from the answers. Pushes back on vagueness. / 逐节评分，点名最弱两节，针对性提问并据回答修补。对含糊表述据理反驳。 | No section below "solid", or user overrides / 无低于「扎实」的章节，或用户强制通过 |
| `finalize` 定稿 | Summary + completeness checklist + suggests snapshotting a revision and submitting for review. / 总结 + 完整度清单 + 建议存版本快照并提交评审。 | Conversation marked `completed` / 对话置为 `completed` |

**Tools (function calling) / 工具（函数调用）**:

- `get_prd()` → current title + sections / 当前标题与章节
- `update_prd_section(section, markdown)` — validated against the 8 keys; each call also emits a `prd_updated` SSE event so the UI refreshes the section live / 校验限定 8 个键；每次调用同时发出 `prd_updated` SSE 事件，UI 实时刷新该节
- `advance_stage(next)` — validated against the legal transition order / 校验合法的阶段顺序

**Memory / 记忆**: full history in `agent_messages`. Context assembly per turn: system prompt (stage) + current PRD sections + rolling summary from `conversation.meta.summary` + last ~20 messages. When history exceeds the threshold, the server summarizes older turns into `meta.summary` with one extra LLM call. No framework memory abstraction — it's one table and one summarize function. **[Implementation note: the shipped coach is last-20-messages only; the `meta.summary` rolling summarization was deliberately deferred (see the ponytail note in `server/internal/agent/coach.go`) and is not yet implemented.]** `// ponytail: naive last-N + summary; add token-budgeted packing if coaching sessions blow the context window`
**记忆**：完整历史在 `agent_messages`。每轮上下文组装：系统提示（按阶段）+ 当前 PRD 章节 + `conversation.meta.summary` 滚动摘要 + 最近约 20 条消息。超阈值时服务端额外调用一次 LLM 将旧轮次压入 `meta.summary`。不用框架记忆抽象——就是一张表加一个摘要函数。

**Prompts / 提示词**: Chinese-primary (mirror the user's language), concise, structured, no fabrication — inheriting the spirit of the existing system prompt (「回答简洁、结构化，不编造信息」). Prompt texts live in `internal/agent/prompts.go` as constants, versioned in git.
**提示词**：中文为主（跟随用户语言）、简洁、结构化、不编造——继承现有系统提示精神。提示词文本作为常量放在 `internal/agent/prompts.go`，随 git 版本化。

## 4. Streaming & Limits / 流式与限流

`POST /api/v1/conversations/{id}/messages` responds `text/event-stream` / 响应为 `text/event-stream`：

```text
event: message_start   data: {"id": "<assistant message uuid>"}
event: delta           data: {"text": "..."}          # token chunks / 逐段文本
event: prd_updated     data: {"section": "goals"}     # after a tool write / 工具写入后
event: stage_changed   data: {"stage": "draft"}
event: message_end     data: {"id": "...", "tokens": 812}
event: error           data: {"error": "...", "code": "..."}
```

Rate-limit and circuit checks run **before** the stream opens, so throttling still returns plain JSON 429/503 with the preserved `APP_THROTTLE` / `CIRCUIT_OPEN` / `ANTHROPIC_RATE_LIMIT` shapes ([02 §3](02-backend-go.md)) — the frontend countdown logic is untouched. Each user message creates an `ai_runs` row (the limiter's counting source); provider 429 now marks the run `failed` (old pending-leak bug fixed).
限流与熔断检查在流打开**之前**执行，被限流时仍返回普通 JSON 429/503（形状保持不变，见 [02 §3](02-backend-go.md)）——前端倒计时逻辑零改动。每条用户消息落一行 `ai_runs`（限流计数来源）；供应商 429 现在会将 run 置为 `failed`（修复旧的 pending 泄漏 bug）。

The agent loop: call LLM with tools → stream text deltas out as they arrive → on tool calls, execute, append tool results, call again → repeat until a plain text turn ends. Tool executions and the loop are **our** code; the LLM library only does one thing: "given messages + tools, stream one model response".
Agent 循环：带工具调用 LLM → 文本增量即到即转发 → 遇到工具调用则执行、追加结果、再调用 → 直到纯文本轮结束。工具执行与循环是**我们自己的**代码；LLM 库只做一件事：「给定消息与工具，流式返回一次模型响应」。

That one thing hides behind one interface in `internal/agent/llm.go`, so the library choice (below) is swappable without touching coach logic:
这件事藏在 `internal/agent/llm.go` 的单一接口之后，因此库的选择（见下）可替换而不动教练逻辑：

```go
type ChatStreamer interface {
    // StreamChat sends messages+tools, invokes onDelta for each text chunk,
    // and returns the final turn (text and/or tool calls).
    // StreamChat 发送消息与工具，逐段回调 onDelta，返回最终轮次（文本和/或工具调用）。
    StreamChat(ctx context.Context, msgs []Message, tools []Tool, onDelta func(string)) (Turn, error)
}
```

## 5. LLM Library — Research Findings & Decision (D6/Q2: langchaingo, decided) / LLM 库——调研结论与决策（D6/Q2：langchaingo，已定）

The requirement names LangChain's Go library. We researched its mid-2026 state; the findings shaped *how* we use it, and the fallbacks below stay documented.
需求点名 LangChain 的 Go 库。我们调研了其 2026 年中的状态；结论决定了*如何*使用它，下列后备方案记录在案。

**langchaingo (`tmc/langchaingo`)** — latest release v0.1.14 (2025-10-20, still pre-1.0); **last commit to main 2026-01-11 — ~6 months of silence**; ~9.5k stars, ~405 open issues/PRs, effectively single-maintainer, and an open issue literally titled *"is this project dead?"* (#1486, 2026-03-26). It does cover our surface (streaming via `WithStreamingFunc`, tools via `WithTools`, Anthropic + OpenAI-compatible providers so DeepSeek/Qwen work via `WithBaseURL`), **but** open bugs sit exactly on our critical path: the Anthropic provider **drops parallel tool calls beyond the first** (2026-01), streaming-reasoning callbacks are reported broken with Claude (2026-04), DeepSeek `reasoning_content` is lost across tool round-trips (2026-02), and Anthropic prompt caching doesn't work (2026-01). Its bundled agent executor is legacy string-input ReAct — we would not use it regardless (our loop in §4 replaces it).
**langchaingo**——最新版 v0.1.14（2025-10-20，仍未到 1.0）；**主分支最后提交 2026-01-11，已静默约 6 个月**；约 9.5k star、约 405 个未关 issue/PR、实质单人维护，且有一个标题就是*「这项目死了吗？」*的未关 issue（#1486）。功能面覆盖我们所需（流式、工具、Anthropic 与 OpenAI 兼容端点，DeepSeek/Qwen 可经 `WithBaseURL` 使用），**但**未修复 bug 恰在关键路径上：Anthropic 提供方**丢弃第一个之外的并行工具调用**、流式推理回调对 Claude 报告失效、DeepSeek `reasoning_content` 在工具往返中丢失、Anthropic 提示缓存失效。其自带 agent executor 是老式字符串输入 ReAct——无论如何我们都不会用（§4 的自建循环取代之）。

**Alternatives / 备选**:

- **`anthropic-sdk-go` (official / 官方)** — v1.58.0 (2026-07-16), near-weekly releases; full streaming, tool use, extended thinking, prompt caching. Claude-only, no agent scaffolding — which we don't need, since §4's loop is ~150–200 lines either way. Lowest-risk if Claude-first.
  官方 SDK——v1.58.0，近乎每周发版；流式、工具、扩展思考、提示缓存俱全。仅 Claude、无 agent 脚手架——而我们本就不需要（§4 循环两种选择下都是约 150–200 行）。Claude 优先时风险最低。
- **`cloudwego/eino` (ByteDance / 字节)** — 12.4k stars, daily activity, v0.9.12 (2026-06); typed graph orchestration, streaming-aware agent runtime with interrupt/resume, and dedicated model components for **claude, deepseek, qwen**, openai, gemini — the best CN-model coverage in Go. Caveats: ByteDance governance, partly Chinese-first docs.
  12.4k star、日更、v0.9.12；类型化图编排、支持中断/续跑的流式 Agent 运行时，且有 **claude、deepseek、qwen** 等专用模型组件——Go 生态中最好的国产模型覆盖。注意：字节治理、文档部分中文优先。
- **Genkit Go (Google)** — stable 1.0 since 2025-09; good structured output + tracing UI; weaker DeepSeek/Qwen story. A credible middle ground, not our first pick.
  2025 年 9 月起稳定 1.0；结构化输出与追踪 UI 好；DeepSeek/Qwen 支持较弱。可信的折中项，非首选。

**Decision (D6, settled) / 决策（D6，已定）**: ~~**build on langchaingo behind the `ChatStreamer` interface**, honoring the requirement, with the bugs engineered around: (a) prompt the model to make **one tool call per turn** and defensively error on parallel calls (sidesteps the known Anthropic parallel-tool-call bug — our staged coach only needs sequential tools anyway); (b) text-only streaming via `WithStreamingFunc` (we don't use the broken streaming-*reasoning* path); (c) pin v0.1.14 and treat langchaingo as frozen upstream — no feature of ours may depend on an upstream fix landing.~~ **Superseded during implementation, per explicit user direction**: langchaingo dropped entirely. Running the real coaching flow against a live extended-thinking model surfaced exactly the kind of bug this section worried about, just a different one — the pinned Anthropic provider maps each response content block to its own `Choices[i]`, so a leading "thinking" block silently ate the real text/tool_use answer at `Choices[0]`. Replaced with a direct client (`internal/agent/anthropic.go`) implementing the Messages API's stream format (message_start/content_block_start/_delta/_stop/message_delta/message_stop) directly over stdlib `net/http` — no dependency at all, `ChatStreamer` untouched. See [BUG-009](../../issues/2026-07-21-bug-009-langchaingo-removed.md). The one-tool-call-per-turn prompting discipline (a) stays, since it's a real property of the coach design, not just a langchaingo workaround. Per Q5, Claude is still the only first-class provider for v1.
**决策（D6，已定）**：~~**在 `ChatStreamer` 接口之后基于 langchaingo 构建**，尊重需求，并以工程手段绕开已知 bug：(a) 提示模型**每轮只发一个工具调用**并对并行调用防御性报错；(b) 仅经 `WithStreamingFunc` 做文本流式；(c) 锁定 v0.1.14、把 langchaingo 当作冻结上游。~~**实现期已被取代，按用户明确指示**：完全去掉 langchaingo。针对真实的、启用扩展思考的模型跑通真实教练流程时，恰好撞上了这一节所担心的那类 bug，只是换了一种：锁定版本的 Anthropic provider 把响应的每个内容块映射成独立的 `Choices[i]`，导致排在前面的 "thinking" 块在 `Choices[0]` 悄悄吃掉了真正的文本/工具调用答案。已替换为直连客户端（`internal/agent/anthropic.go`），直接在标准库 `net/http` 之上实现 Messages API 的流式格式（message_start/content_block_start/_delta/_stop/message_delta/message_stop）——零依赖，`ChatStreamer` 不变。见 [BUG-009](../../issues/2026-07-21-bug-009-langchaingo-removed.md)。「每轮一个工具调用」的提示纪律 (a) 保留，因为这是教练设计本身的真实属性，不只是绕开 langchaingo 的权宜之计。按 Q5，v1 仍仅 Claude 一等支持。

## 6. Testing the Agent / 智能体测试

Per the no-mock rule, agent tests hit the real provider, but sparingly / 按无 mock 规则，测试打真实供应商，但克制用量：

1. Tool plumbing test (no LLM): call the tool executor directly — section validation, stage transition legality, `prd_updated` event emission. / 工具管线测试（不经 LLM）：直接调用工具执行器——章节校验、阶段转移合法性、事件发出。
2. Real-API smoke (gated by `ANTHROPIC_AUTH_TOKEN`): one clarify-stage exchange asserts a streamed non-empty reply; one draft-stage exchange asserts at least one `update_prd_section` call lands in `prds.sections`. / 真实 API 冒烟（`ANTHROPIC_AUTH_TOKEN` 门控）：澄清阶段一次往返断言非空流式回复；起草阶段一次往返断言至少一次 `update_prd_section` 落入 `prds.sections`。
3. Template-reply mode (AI config absent) is itself tested — it's the offline dev path and must keep working. / 模板回复模式（无 AI 配置）本身也要测——它是离线开发路径，必须常青。
