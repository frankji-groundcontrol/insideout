# InsideOut — Product Experience Baseline

> **Authority:** canonical target product experience, confirmed through the
> product interview completed on 2026-08-13.
>
> This document describes what InsideOut should become, not what every current
> screen already does. Architecture documents describe the running system;
> historical plans describe decisions at their time. When an older product
> assumption conflicts with this baseline, this document wins.

## Product thesis

InsideOut helps the Driver of a software idea or project—and the people and
Agents collaborating with them—turn uncertainty into a series of workable,
time-bound versions, then carry those versions through delivery and real-world
validation.

Its promise is not a perfect idea or a perfect PRD. There is only movement
towards perfection, never a final state called perfect. Without a version,
people remain stuck in the Idea stage; InsideOut's first responsibility is to
help them create something concrete enough to inspect, challenge, change, and
act on.

The product connects one continuous loop:

`Idea or existing project → working version → human Commit → timed Roadmap → Git evidence → release → validation → next version`

## Who InsideOut serves

InsideOut currently serves two usage contexts while its first commercial
segment is still being validated:

- a Driver—founder, product lead, or team lead—responsible for turning
  uncertain software work into a decision, plan, and shipped result with
  collaborators;
- a time-boxed workshop or cohort in which participants need guidance from a
  raw Idea to a version that can be reviewed and acted on.

The team lead's need for visibility without chasing remains important, but it
is a Lens over shared work rather than a separate product or source of truth.

## Product principles

1. **A workable version beats an endless idea.** Form a version as soon as the
   critical questions are answered. Unknowns stay visible as assumptions or
   open questions; they do not silently become facts.
2. **Truth over document completion.** The Coach may recommend continuing,
   validating, pausing, or abandoning a direction. A human always decides.
3. **One truth, many views.** PRD audiences, Roadmap lenses, Web, GitHub, CLI,
   MCP, and Agents project the same facts and decisions; they do not create
   drifting copies.
4. **Time makes work real.** Work without a time commitment can remain in the
   future, but it cannot enter Now.
5. **Evidence is not outcome.** A commit can show implementation activity. It
   cannot prove release, adoption, or user value.
6. **Human authority stays explicit.** AI can summarize, challenge, and
   propose. Humans Commit product versions, Merge product branches, accept
   strategic changes, and confirm product outcomes.
7. **Explain the question.** The Coach shows who a question serves, why it is
   being asked now, and whether it blocks the current version.
8. **Progressive depth.** Show the Why first and reveal implementation detail
   only when the reader, role, or task needs it.

## Canonical journey

InsideOut has two valid starting points:

- **A new Idea.** Capture remains one-step and private to its author by
  default. Choosing to start progressing creates the Project, preserves the
  original title and body as coaching context, and opens the first working
  version and Roadmap planning workspace. The Roadmap becomes an authoritative
  baseline only when tied to a human-confirmed PRD Commit.
- **An existing project or GitHub repository.** The user can connect work that
  has already started and establish its current product and delivery baseline
  without inventing a retrospective Idea.

Both entries converge on the same PRD, version, Roadmap, role, and evidence
model. Creating a Roadmap is planning, not the start of development. The
development boundary is crossed when a person or Agent begins an executable
code item as their main Focus within the team's Now work; at that point a Git
repository is required.

## The Coach and the first workable version

### Coaching contract

The Coach is a constructive, truth-seeking partner rather than a template
filler. It adapts interview depth to the problem's complexity; it does not
force a fixed number of questions or make the user complete a fixed sequence
of sections before seeing useful output.

Every question carries three pieces of context:

- priority: **must clarify now**, **should clarify this version**, or
  **validate later**;
- the reader, persona, or decision the answer primarily serves;
- why this question matters at this moment.

Questions may be grouped when independent. The Coach asks one question at a
time only when the answer changes the next branch of the interview.

The user can always choose **form a version now**. The Coach then states what
is missing, marks the gaps as assumptions or open questions, and produces an
editable version. No magic phrase is required.

### Critical product argument

Before recommending a first version, the Coach seeks enough clarity to make a
coherent product argument:

- What is this, and what problem or change does it address?
- Why this direction rather than the current alternative?
- Why this person or team?
- Why now?
- What is changing in the present era?
- What future does this product bet on?

These are critical questions, not mandatory headings. Their depth and wording
depend on the product and intended reader.

### Concrete people and paths

A version describes concrete people rather than one abstract "user." Personas
are prioritized as core users, key participants, and later users. Each persona
records:

1. how the person arrives or what triggers them;
2. what they do;
3. what value or change they receive;
4. the state in which they leave, the next step, and why they may return.

The model also captures value exchanges and handoffs between personas. Every
persona is labeled as grounded in real evidence, synthesized from evidence, or
an assumption. A plausible persona must never masquerade as a researched one.

### Critique and completion

Critique begins with a severity-ranked overview. Blocking and major findings
are handled individually with a recommendation and a keep-as-is option. Minor
findings are folded by default and can be accepted or ignored in a batch; they
do not block a version.

A version may be complete for its current purpose while still carrying
assumptions and validation questions. Only blockers require resolution or an
explicit human override. Independent AI review may fail or be unavailable;
that is disclosed and can be retried, but it does not trap the user in an
unfinished session.

The user can correct the Coach at any time by editing directly or continuing
the conversation.

## One PRD core, multiple audience views

The product distinguishes a **persona**, who uses the product being designed,
from an **audience**, who reads or acts on the PRD.

Each working version has one human-confirmed primary audience. The Coach may
recommend it, and secondary audiences may be attached. The same fact and
decision core can then be projected as:

- **Decision view** for a boss or decision maker: why, why now, why this team,
  investment and return, risks, and decisions required.
- **Management view** for a product manager and team: personas, paths,
  priorities, scope, dependencies, assumptions, and sequence.
- **Delivery view** for engineering and QA: behavior, states, rules,
  boundaries, acceptance, dependencies, and evidence.
- **Co-creation and validation view** for early users, friends, and advisers:
  the thesis, usable material, uncertain assumptions, requested feedback, and
  unresolved choices.

These are projections, not separately maintained documents. Readiness is also
audience-specific: ready for validation, decision, management, or delivery.
There is no misleading universal "100% complete" score. Authorized readers
can switch audience views without gaining any additional permission.

## Facts, evidence, and privacy

The Coach records candidate facts without interrupting the user after every
sentence. Before a meaningful Commit, it presents the critical facts together
for confirmation.

The internal evidence experience shows:

- the Coach's interpretation;
- the supporting user words or source;
- whether it is confirmed, synthesized, assumed, or still to validate;
- small actions to confirm, correct, remove, or discuss it.

This is a trust layer, not a second generic form system. Raw quotes and private
conversation remain internal by default. Audience views show conclusions and
only expose raw material when a person deliberately shares it. Facts are not
silently reused across PRDs or Projects; intentional reuse can be introduced
only with clear user control. Candidate facts belong to the current PRD and
Product Branch by default so competing hypotheses cannot silently borrow one
another's evidence.

## Product version control

InsideOut teaches Git-shaped product thinking through accessible language:

- **Working version:** autosaved, mutable work; autosave is not a formal
  product version.
- **Commit / Save version:** an immutable, meaningful checkpoint confirmed by
  a human. It records a name, primary audience, change summary, unresolved
  validation items, and a Diff from the preceding Commit.
- **Product Branch / Explore direction:** an alternative product hypothesis,
  such as a different core user, value proposition, business model, or scope.
- **Diff / Compare changes:** what changed and why, not merely a text diff.
- **Merge / Combine conclusion:** a human decision to bring a Product Branch
  into the main product line.

An authorized Driver can Commit a working checkpoint at any time. The Coach
recommends useful Commit moments—for example, after the product argument, core
path, or a readiness milestone—but never gates or performs a human-initiated
Commit. Saved versions are immutable; further changes form the next working
version.

A Decision Log accompanies meaningful Commits, Merges, overrides, and reversed
decisions. It records who decided, why, the evidence considered, and the
alternatives not chosen. Any Commit can begin a new working version or Product
Branch without rewriting history.

A Product Branch is not a Git branch, and an audience view is not a Product
Branch. Once a product direction is merged, its delivery may link to one or
more Git branches.

## Collaboration and authority

Workspace roles manage organization access. Product authority is assigned per
Project, because the same person may play a different part on different work:

- **Decision Owner:** confirms the mainline, Product Branch Merges, product
  baselines, and product outcomes.
- **Driver:** leads the coaching flow and working version and can Commit
  working checkpoints. The Decision Owner confirms milestone and mainline
  versions.
- **Contributor:** adds material, alternatives, and proposals.
- **Reviewer:** comments and requests changes without directly changing the
  mainline.
- **Viewer:** reads the permitted views.

Each Product Branch has one current Driver, and that responsibility can be
transferred. Contributions from other people or Agents remain proposals until
accepted into the shared facts or version.

Each Now execution item has exactly one human responsible owner. Multiple
people or Agents may Focus on it, but an Agent cannot be its only owner. Focus
is a non-exclusive pointer for one person or one Agent session, with one main
Focus per session; team Now is the shared set that aggregates active Focuses.
GitHub or an Agent may advance leaf-level delivery evidence when acceptance is
explicit; the Driver confirms milestones, and the Decision Owner confirms
product results and Product Branch Merges.

Early users and advisers can receive a scoped feedback invitation without
joining the Workspace. Feedback retains author, time, and context and becomes
a suggestion or proposed branch rather than a silent mainline edit. External
respondents see their own feedback by default, not everyone else's; the team
sees the complete set, and the Decision Owner can publish a deliberate
synthesis.

## The shared Roadmap Graph

The Roadmap answers three questions: **Where are we? What is done? What is
next?** It is one shared, versioned, hierarchical graph tied to an explicit PRD
Commit baseline.

The graph may express outcomes, user journeys or capabilities, verifiable
milestones, deliverables, execution work, and evidence. It does not create
empty levels merely to satisfy a schema. Moving upward reveals Why; moving
downward reveals acceptance, work, and evidence.

The default Progress view makes Now dominant, shows at most three justified
Next items, and presents Done with its evidence. The full Canvas reveals the
larger hierarchy and execution frontier. History shows versions, decisions,
and evidence. These are views of one graph, not separate stores.

Different people use different Lenses over that graph:

- **Decision:** outcomes, risk, time, and decisions required.
- **Product:** persona paths, priorities, dependencies, blockers, and next
  milestones.
- **Delivery:** acceptance, work items, code review, tests, and deployments.
- **Validation:** demos, hypotheses, requested feedback, and evidence.
- **Agent context:** the current objective, Why, baseline, acceptance,
  constraints, dependencies, and latest checkpoint.

Role, Lens, permission, and Agent mode are separate. A role chooses a sensible
default Lens; switching an authorized Lens does not grant new permissions.

### Time is a first-class constraint

Product outcomes carry target time ranges. Milestones and execution items
carry explicit deadlines. An item without a deadline cannot enter Now.

Deadline pressure becomes increasingly visible as time closes:

`normal → near → high risk → overdue`

Overdue work is never silently closed. A human can continue it, change the
deadline, transfer it, or abandon it, and records the reason. A dependency
delay shows the affected path and a proposed time Diff; a human confirms the
rebaseline. Agents include deadline pressure when recommending priority and
next work.

## Git, CLI, MCP, and Agents

InsideOut recommends connecting GitHub immediately after an Idea is saved and
when a Project is created manually, but the prompt remains skippable through
coaching, research, and planning. Git becomes required only when a person or
Agent begins executable code work as their main Focus within team Now.

Matched GitHub activity automatically advances leaf-level delivery evidence on
the Roadmap:

- a commit may show implementation activity;
- an opened pull request may show review;
- a merged pull request may show implementation completion;
- a deployment may show release.

User feedback and metric observations arrive through their own evidence
sources and may show validation; they are never presented as GitHub events.

No Git event silently proves a higher-level product outcome. Unmatched
activity stays visibly unassociated for human review instead of being guessed
onto a Roadmap item. Reverts and failed deployments append evidence; they do
not erase history. When delivery reveals a product change, a person or Agent
creates a Change Proposal that shows the impact on views, journeys, Roadmap,
and existing work. A human accepts it before the baseline changes.

Web, GitHub, CLI, MCP, and Agents operate on the same source of truth. The
minimum agent-facing vocabulary is `context`, `focus`, `checkpoint/report`,
`propose`, and `version`.

An Agent receives compact, Focus-scoped context rather than the entire graph:

- brainstorming mode emphasizes personas, hypotheses, decisions, and open
  questions;
- implementation mode emphasizes the work item, acceptance, dependencies,
  deadline, and Git context;
- review mode emphasizes the relevant review package and baseline.

Agents can checkpoint work and propose structure, scope, or priority changes.
They cannot directly apply strategic changes, Commit a product version, or
Merge a Product Branch.

## Completion and learning

InsideOut distinguishes **implemented**, **released**, and **validated**.
Completion links to verifiable evidence and may be overridden by a human only
with a reason.

Release is not the end of the product loop. The team compares real outcomes
with the committed personas, assumptions, and success criteria, updates the
evidence, and begins the next version or Product Branch. Progress is a chain of
meaningful versions, not a claim that the idea has become perfect.

## Minimum coherent end-state loop

The target is coherent only when one Driver and a small team can eventually
complete one real loop:

1. begin from a private Idea or existing GitHub project without losing its
   original context;
2. answer prioritized, explained, complexity-adaptive coaching questions or
   form a version now with explicit gaps;
3. produce an editable product argument with prioritized concrete personas and
   paths;
4. choose an audience and view one fact core through the relevant projection;
5. confirm an immutable Commit with a Diff, change summary, open validation
   items, and Decision Log;
6. establish a Roadmap baseline with Now, justified Next, evidenced Done,
   dependencies, one human owner, and deadlines;
7. separate Decision Owner and Driver authority, with strategic baseline and
   Merge decisions remaining human-confirmed;
8. require Git when code work becomes someone's main Focus within team Now and
   automatically associate matched GitHub evidence without claiming unproven
   user value;
9. give Web and one Agent interface the same Focus-scoped Roadmap context;
10. distinguish implementation, release, and validation and start the next
   version without erasing history.

This is an end-state acceptance loop, not one implementation batch. Delivery
must divide it into the smallest independently useful vertical slices.

Deliberately excluded from this end-state loop: copied PRDs or Roadmaps per role,
a custom role builder, automatic AI Commit or Merge, formal versions for every
autosave, a generic evidence CRUD system, silent cross-PRD memory, full-graph
Agent prompts, multi-provider Git abstractions, and productivity scoring from
commit volume.

## Derived delivery defaults

The interview confirmed the experience above. The following are the simplest
delivery defaults implied by it, not separate product commitments:

- Start with GitHub and one primary repository per Project. Support private
  repositories before enforcing the development gate for a team that uses
  them; add providers or multiple repositories only when real use requires it.
- Default the first-run Lens to the Driver/Product view and let role and task
  select later defaults.
- Make the lead's cross-project home exceptions-first: overdue or high-risk
  work, blocked decisions, near milestones, and the last meaningful evidence,
  not raw commit volume.
- Treat the existing Progress timeline as the History/Activity Lens of the
  Roadmap Graph, not a second progress truth.

These defaults should change only when observed usage demonstrates a need.

One product assumption remains deliberately open: the first commercial core
persona has not been narrowed beyond a Driver responsible for moving software
work from uncertainty to delivery. Founder, product lead, team lead, and
workshop Driver must not be treated as equally validated personas; choose and
validate the initial segment before optimizing go-to-market onboarding around
one of them.

## Superseded assumptions and current status

This baseline supersedes the earlier product assumptions that every PRD is
eight fixed sections, every coaching session follows four rigid stages, a
Workspace admin approves all PRDs, captured Ideas are automatically visible to
the Workspace, Roadmap status alone proves completion, or a manually synced
public-commit timeline is an independent source of progress truth.

The current Go, PostgreSQL/RLS, Flutter, PRD Coach, Roadmap Canvas, and public
GitHub-sync implementation remains the running foundation. It does **not** yet
implement this whole target experience. Delivery should evolve that foundation
in small vertical slices and reuse its existing PRD revisions, Roadmap tree,
project updates, and GitHub integration before adding new mechanisms.

## Brand commitments

- **Name:** InsideOut.
- **Voice:** warm, confident, concrete; English and Simplified Chinese are
  first-class product languages.
- **Visual identity:** the existing Ink & Seal system—celadon ground, sumi-ink
  text, a single vermilion accent, Noto Serif SC display, referenced (never
  bundled) PuHuiTi sans, and deliberate light and dark themes.
