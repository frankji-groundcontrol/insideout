# 12 · First-Run User

**Lens:** New-user journey: landing, auth, first workspace, first three minutes
**Score:** **4 / 10**
**Bottom line:** The landing page and ideas wall show a coherent, trustworthy product with a clear story, but the first three minutes derail immediately after login: the floating dock renders at mid-viewport on every screen (root cause: an expanding Center inside Scaffold.bottomNavigationBar), which also pushes the create-project FAB off-screen, several dock buttons are silent dead clicks, the project page is headlined '证据' instead of the project name, and workspace/project/PRD surfaces read as empty shells. A new user can register and capture an idea, but cannot reliably navigate the loop the landing page just promised. The visual identity is strong; the chrome and wayfinding are not.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

Walked the new-user path in order against the capture set in `shots/` — landing (01), login modal (02), register modal (03), dashboard light/dark (04/05), workspace (06/12), project (07/13), ideas wall (08/14), PRD desk (09/15), roadmap (10/16), versions (11/17), profile (18), plus the narrow shots (19/20) — and read the code behind each beat: `landing_page.dart`, `auth_door.dart`, `auth_form.dart`, `dashboard_page.dart`, `app.dart`, `product_dock.dart`, `workspace_page.dart`, `project_page.dart`, `ideas_page.dart`, `prd_desk.dart`, `coach_panel.dart`, `versions_stack.dart`, and the zh/en catalogs.

## What is good

- The landing hero "把想法，一件件打磨出来。" plus a one-sentence subtitle precisely explains the idea→PRD→roadmap→ship loop, with a 4-step assembly story below (`client/lib/features/landing/landing_page.dart:14-19`, `shots/01-landing-logged-out.png`).
- The ideas wall's capture input is front-and-center, and the privacy hint "仅自己可见——想法默认只有作者能看到（工作区管理员除外），转化前不会分享。" (`client/lib/l10n/zh.dart:192`, `shots/08-ideas-wall-dark.png` / `shots/14-ideas-wall-light.png`) is exactly the trust copy a first-run user needs before typing an unpolished thought.
- Light/dark pairs (04/05, 06/12, 07/13, 08/14, 09/15, 10/16, 11/17) are visually coherent with no token breakage; the zh/en catalogs are at exact parity (347 keys each).
- The versions page has per-audience tabs (验证/决策/管理/交付), a concrete readiness checklist, and a clear 提交版本 CTA (`shots/11-versions-dark.png` / `shots/17-versions-light.png`) — the strongest first-run page after the ideas wall.
- The roadmap empty state tells the user what to do next: "还没有路线图。先加一个顶层目标，再拆成可并行的分支。" (`client/lib/l10n/zh.dart:151`).
- The register modal states the password rule "至少 8 个字符" inline before submission (`client/lib/features/auth/auth_form.dart:208`, `shots/03-register-modal.png`).

## Findings, worst first

### dock-pill-mid-screen-root-cause — Critical

The dock is a `Center` inside `Scaffold.bottomNavigationBar`, so the nav bar expands to full height and the pill renders at the viewport's vertical center, overlaying body content on every signed-in page: the dashboard chapter paragraph is clipped mid-sentence (04/05), the workspace delete row is half-hidden (06/12), and the versions checklist's last two items are unreadable (11/17). Pages pad `bottom:140` expecting a bottom dock that never materializes at the bottom.

**Evidence:** `client/lib/chrome/product_dock.dart:26`

```dart
    return SafeArea(
      minimum: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      child: Center(
        child: Material(
... (client/lib/app.dart:135) bottomNavigationBar: signedIn ? const ProductDock() : null,
... (client/lib/app.dart:71)      extendBody: true,
... (client/lib/chrome/product_dock.dart:15)  static const height = 88.0;  // never referenced anywhere
```

**Fix:** Bound the nav bar: wrap `ProductDock` in `SizedBox(height: ProductDock.height)` (or replace `Center` with `Align(alignment: Alignment.bottomCenter)`), drop `extendBody` or keep it with a real bottom inset, and have `AppScaffold` own a single bottom-safe-area constant that body pages consume instead of ad-hoc bottom padding.
**Verdict:** CONFIRMED

### create-project-fab-invisible — High

The create-project FAB is the only way to create a project, and it is not visible anywhere on the workspace page (06/12) — pushed off-position by the full-height `bottomNavigationBar` from the finding above. A brand-new user who just joined a workspace sees one project card and four label rows; the happy path the landing page promised (idea → PRD → roadmap) cannot be started in a fresh workspace.

**Evidence:** `client/lib/features/workspace/workspace_page.dart:115`

```dart
          fab: FloatingActionButton(
            onPressed: _createProject,
            child: const Icon(Icons.add),
          ),
... (client/lib/app.dart:133-134) floatingActionButton: fab,
      floatingActionButtonLocation: FloatingActionButtonLocation.endFloat
Screenshot 06-workspace-dark.png and 12-workspace-light.png: no FAB in any corner; bottom-right is empty.
```

**Fix:** Fix the dock height (finding above) so the endFloat FAB returns; additionally surface an explicit 'new project' button in the projects section so creation does not depend on a lone FAB. Verify by loading /workspace/<id> and asserting a visible create affordance.
**Verdict:** CONFIRMED

### project-page-evidence-heading — High

The project page leads with a giant display heading "证据" (Evidence) and demotes the actual project name below it, so a user who clicked "Onboarding Revamp" lands on a page whose biggest word is "证据" — they cannot confirm which project they are in, and the breadcrumb-less navigation compounds it.

**Evidence:** `client/lib/features/project/project_page.dart:150`

```dart
                    Text(
                      l10n.t('evidence.title').toUpperCase(),
                      style: theme.textTheme.displaySmall,
                    ),
                    const SizedBox(height: 8),
                    Text(title, style: theme.textTheme.headlineSmall),

Screenshot 07-project-dark.png / 13-project-light.png: '证据' at ≈(75,95) is the largest text on screen; 'Onboarding Revamp' sits under it.
```

**Fix:** Make the project title the display heading; demote 证据 to an eyebrow/section label over the updates/evidence feed, or retitle the page with project-specific copy.
**Verdict:** CONFIRMED

### dock-dead-clicks-evidence-version — Medium

Two of six dock items silently do nothing in common states: 证据 is a guaranteed no-op (`stay()`), and 版本 is a dead click on the PRD desk because it navigates to the page already open. A first-run user tapping either gets zero feedback and concludes the app is broken; the six-item dock — the app's primary navigation — teaches them that clicks are unreliable.

**Evidence:** `client/lib/chrome/product_dock.dart:129`

```dart
      case DockItem.evidence:
        stay();
... (client/lib/chrome/product_dock.dart:109-112) case DockItem.version:
        final prd = focus.prdId;
        if (prd != null) {
          context.go('/prd/$prd');

Dock walk in captures.md: '证据 — Dead click from project roadmap — no navigation, no feedback (A7)'; '版本 — From /prd desk → dead click'.
```

**Fix:** For 证据: navigate to /projects/<id> only when not already there, else scroll/highlight the evidence section; for 版本: go to /prd/<id>/versions (same target as 提交) or render the current item disabled instead of silently re-pushing the current route.
**Verdict:** DOWNGRADED (was High) — `stay()` does navigate to /projects/<id> when a project is known (`product_dock.dart:93-94`, route `router.dart:37`). Dead clicks occur only in a fresh session at /dashboard with null focus; the Version dead click is self-navigation of the already-selected item.

### auth-card-untitled-header-on-scrim — Medium

The auth modal card has no title on it — the cobalt disc, "InsideOut" wordmark, and greeting (the only 登录/创建账号 labels) render outside the card as siblings above it in a centered Column, floating on the dimmed landing. This is the code-level why behind the confirmed shot-02 collision (wordmark overlapping 开始打磨/登录), and even when it doesn't collide the greeting is 12px bodySmall dark text on a black54 scrim, so the modal is effectively untitled for first-run users and screen-reader context.

**Evidence:** `client/lib/features/auth/auth_door.dart:50`

```dart
                          InkReveal(
                            child: Container(
                              width: 56,
                              height: 56,
                              decoration: const BoxDecoration(color: Cobalt.hue500, shape: BoxShape.circle),
                            ),
                          ),
                          const SizedBox(height: 20),
                          InkReveal(
                            ...
                            child: Text('InsideOut', style: theme.textTheme.headlineSmall),
...
(child: Text(greeting, style: theme.textTheme.bodySmall), — auth_door.dart:69)

Screenshot 02-login-modal.png: card contains only 邮箱/密码 fields and a 登录 button; the wordmark floats above the card over the hero text.
```

**Fix:** Move the disc + wordmark + greeting inside the raised Material card as its header, deleting the floating scrim stack; keep Esc/scrim-tap close behavior.

### idea-card-body-not-tappable — Medium

Idea poster cards look like the primary object but are not tappable — only the small 'PRD' text button or the '…' menu does anything. A new user's first idea is the natural thing to click; nothing happens, so they hunt for the tiny PRD chip. The most prominent interactive-looking element on the wall is inert.

**Evidence:** `client/lib/features/workspace/ideas_page.dart:278`

```dart
    return FieldCard(
      color: palette.surfaceRaised,
      padding: EdgeInsets.zero,
      child: Container(
        constraints: const BoxConstraints(minHeight: 240),
...
(child: TextButton(
  onPressed: idea.prdId == null ? onConvert : onOpenPrd,
  child: Text(idea.prdId == null ? l10n.t('idea.convert') : l10n.t('prd.title')), — ideas_page.dart:334-337)

Screenshot 08/14 ≈(255,457): clicking the title/body is a dead click (captures.md A6d).
```

**Fix:** Give the poster FieldCard an onTap that opens the PRD (or starts conversion when none exists), with the menu chip retained for edit/drop.

### workspace-rows-empty-shells — Medium

想法/设置/编辑/删除这个工作区 are label-only rows with no chevron, no description, and the destructive delete row styled identically to neutral rows. First-run users cannot tell the rows navigate anywhere, may confuse 设置 with 编辑, and the workspace-deleting row looks exactly like the others — a misclick away from a destructive, confirm-dialogged action presented with zero warning color.

**Evidence:** `client/lib/features/workspace/workspace_page.dart:160`

```dart
                      FieldCard(
                        onTap: _delete,
                        child: Text(
                          l10n.t('workspace.deleteWorkspace'),
                          style: theme.textTheme.titleMedium,
                        ),
                      ),

FieldCard (client/lib/theme/field_card.dart:22-40) has no chevron/trailing affordance of any kind. Screenshot 06/12 ≈(720,192)-(720,443): four identical rows; 设置 vs 编辑 distinction is unexplained.
```

**Fix:** Add trailing chevrons and one-line descriptions; render the delete row in fgDanger and visually separated (or behind the settings page).

### create-workspace-unreachable-nonempty — Medium

The create-workspace button only exists in the empty state; once a user has one workspace, the dashboard offers join-only, and onCreate is never wired in the non-empty branch. A user in "InsideOut Demo Team" who wants their own workspace has no UI path to create one — the dialog exists (`dashboard_page.dart:35-54`) but is unreachable.

**Evidence:** `client/lib/features/dashboard/dashboard_page.dart:231`

```dart
    if (workspaces.isEmpty) {
      return FieldCard(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.t('workspace.empty'), style: theme.textTheme.bodyLarge),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              children: [
                FilledButton(onPressed: onCreate, child: Text(l10n.t('workspace.create'))),
...
(dashboard_page.dart:249-289: the non-empty Column uses onJoin/onOpen only; the onCreate field is dead in that branch)
```

**Fix:** Keep 创建工作区 visible next to 用邀请码加入 in the non-empty state (a Wrap of two outlined buttons above the workspace list).

### raw-kind-and-iso-timestamps — Medium

The updates feed shows the raw English enum "progress" as the kind chip and a raw ISO-8601 timestamp, while localized keys and friendly time exist elsewhere in the same app — the workspace card two screens away renders "进展 · 8 小时前" from the same data. The zh UI suddenly shows English enum names and machine timestamps on the two screens a new user reads most, so the product feels half-localized and unfinished.

**Evidence:** `client/lib/features/project/project_page.dart:304`

```dart
                            ShadeChip(label: u.kind, tone: ShadeTone.neutral),
                            const SizedBox(height: 8),
                            Text(u.content, style: theme.textTheme.titleMedium),
                            Text(u.createdAt, style: theme.textTheme.bodySmall),

'project.updateKind.progress': '进展', exists at client/lib/l10n/zh.dart:140 and is used at workspace_page.dart:252; timeAgo is used at workspace_page.dart:274.
Screenshot 07/13 ≈(120,588): chip 'progress' + '2026-10-06T00:31:05+08:00'; 09/15 ≈(120,232): 'v1 · 2026-10-06T00:31:05+08:00' (prd_desk.dart:74).
```

**Fix:** Use `l10n.t('project.updateKind.${u.kind}')` and parse createdAt into timeAgo()/isoDate(); do the same for prd_desk.dart's updatedAt composite.

### versions-readiness-mixed-language — Medium

The readiness gap checklist concatenates a localized section name with the server's raw English reason, producing half-Chinese lines like "背景与问题陈述 — the why-this, why-now argument a decision maker needs". A zh-first user on the commit-readiness screen — the page the dock nudges them toward — must read English glosses to understand what is missing.

**Evidence:** `client/lib/features/prd/versions_stack.dart:184`

```dart
                    Expanded(child: Text('${l10n.t('prd.sections.${g.section}')} — ${g.reason}', style: theme.textTheme.bodyMedium)),

Screenshot 11-versions-dark.png / 17-versions-light.png: six rows of zh section name + en gloss; only the 背景与问题陈述 row is unchecked.
```

**Fix:** Localize gap reasons: either accept a locale parameter server-side, or map known reason identifiers to l10n keys client-side and fall back to the raw string only for unknown ids.

### prd-section-cards-blank-no-placeholder — Medium

PRD section editors are borderless TextFields with no hintText, so empty sections render as heading-only cards that give no hint they are editable or what to write. The first-run flow lands here right after converting an idea; the coach asks questions on the right while the left shows three big empty cards — the user has no idea answers belong in those cards or what shape they should take.

**Evidence:** `client/lib/features/prd/prd_desk.dart:96`

```dart
                  TextField(
                    controller: sectionCtrls[key],
                    maxLines: null,
                    minLines: 3,
                    style: theme.textTheme.bodyLarge,
                    decoration: const InputDecoration(
                      border: InputBorder.none,
                      enabledBorder: InputBorder.none,
                      focusedBorder: InputBorder.none,
                      filled: false,
                    ),
                  ),

Screenshot 09/15: 背景与问题陈述 / 目标用户与画像 / 目标与成功指标 are all heading + blank space (captures.md A14).
```

**Fix:** Add per-section hintText (l10n) and a subtle editable affordance (hover tint / '编写…' placeholder); consider seeding one example line for the demo PRD.

### unfinished-dock-label-overlaps-icon — Low

The "unfinished" nudge replaces the 提交 button's label with the longer "未完成：提交", which overflows its slot and renders as a white pill on top of the icon — designed behavior that reads exactly like a stuck tooltip. The label swap also widens the button and shifts neighbors.

**Evidence:** `client/lib/chrome/product_dock.dart:43`

```dart
                    unfinished: item == unfinished && item != current,
                    label: l10n.t(_labelKey(item)),
                    unfinishedLabel: l10n.t('dock.unfinished', {'step': l10n.t(_labelKey(item))}),
...
Text(
  unfinished ? unfinishedLabel : label,
...
'dock.unfinished': '未完成：{step}', (zh.dart:299)

Screenshot 10-roadmap-dark.png ≈(735,443) and 18-profile-light.png ≈(735,443): white pill '未完成；提交' sits on the 提交 icon; absent in 16 (state flipped between passes, so it is designed behavior with broken layout, not a stuck overlay — correcting captures.md A5).
```

**Fix:** Keep the short label and add a non-overlapping indicator (warn-colored dot above the icon, or a small badge offset), with the explanation in the Tooltip only.

## Recommendations

- Fix the dock first: one layout bug (Center in bottomNavigationBar) is responsible for the content overlap on every page, the mid-screen pill, and the missing project FAB; pages already carry bottom:140 padding that will finally work once the dock is truly 88px at the bottom.
- Add a sim-script assertion to `scripts/sim` that walks register → capture idea → convert → PRD desk → versions and fails on dead clicks (evidence/version dock items, idea card body).
- Audit raw-string rendering as a class: snap.error on `dashboard_page.dart:98` and `versions_page.dart:73`, u.kind/u.createdAt on `project_page.dart:304-307`, idea.status fallback at `ideas_page.dart:326`, and g.reason at `versions_stack.dart:184` — wire localizedError/timeAgo/existing l10n keys everywhere.
- Product pass on the dashboard: label the workspace card as a workspace, demote the chapter card or move it below the index, and keep creation (workspace + project) affordances always visible.
- Revisit manifesto-as-UI copy placements (idea.noFolders "没有标签，也没有文件夹。" mid-wall at `ideas_page.dart:251`, home.loopTrail, coach.pullQuotes "大段引语" under the coach title at `coach_panel.dart:142`, and the dangling colon in workspace.inviteHint "把这个邀请码分享给队友：") — either give them context or move them to empty states/tooltips.
- Restyle destructive rows (workspace delete, project delete) with the danger token and add chevrons to all navigable FieldCards so affordance is consistent with the dashboard's '>' cards.

## What this role would do first

Bound the dock: wrap `ProductDock` in `SizedBox(height: ProductDock.height)`, swap `Center` for `Align(alignment: Alignment.bottomCenter)` in `client/lib/chrome/product_dock.dart`, and confirm at 1440x900 and 390x844 that the pill sits at the bottom — this unblocks the invisible create-project FAB and the content overlap in one change.
