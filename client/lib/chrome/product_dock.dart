import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../session/appearance.dart';
import '../theme/cobalt.dart';
import '../theme/cobalt_theme.dart';
import '../widgets/pressable_block.dart';
import 'loop_focus.dart';

enum DockItem { capture, ideas, version, commit, roadmap, evidence }

/// Where a dock tap can go right now. A null path means the item lacks the
/// id it needs and renders disabled instead of bouncing to the dashboard;
/// `pulse` means the destination is already on screen and the tap asks the
/// page for a scroll-highlight.
typedef _DockTarget = ({String? path, bool pulse});

/// Lock chrome E — floating block dock. No left rail at any breakpoint.
///
/// Items that lack the id they need (cold focus) render disabled rather
/// than silently re-routing to the dashboard. Below [_labeledMinWidth]
/// items collapse to icons so all six stay visible and tappable; labels
/// live on in tooltips and semantics.
class ProductDock extends StatelessWidget {
  const ProductDock({super.key});

  static const height = 88.0;

  /// Room a scrolling body leaves below its content so the last row clears
  /// the floating dock. Pages pad with this instead of guessing the sum.
  static const clearance = height + Cobalt.space10 + Cobalt.space4;

  /// The labeled dock needs about this much width under worst-case label
  /// metrics (the en "Unfinished: Commit" swap); below it items show icons
  /// only so the body can never blow out a narrow surface (W3-N1).
  static const _labeledMinWidth = 840.0;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final current = _current(GoRouterState.of(context).uri.path);
    final focus = context.watch<LoopFocus>();
    final unfinished = _firstUnfinished(focus);
    final palette = Cobalt.of(context);
    final brightness = Theme.of(context).brightness;

    return SafeArea(
      minimum: const EdgeInsets.fromLTRB(
        Cobalt.space4,
        0,
        Cobalt.space4,
        Cobalt.space4,
      ),
      child: Align(
        alignment: Alignment.bottomCenter,
        heightFactor: 1.0,
        child: LayoutBuilder(
          builder: (context, constraints) {
            final narrow = constraints.maxWidth < _labeledMinWidth;
            // Narrow cap: a long unfinished label cannot stretch the pill
            // to the viewport edge. Below 840px the items are icons only.
            return ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: narrow ? 360 : double.infinity,
              ),
              child: Container(
                decoration: BoxDecoration(
                  color: palette.brand,
                  borderRadius: BorderRadius.circular(Cobalt.radiusPill),
                  border: Border.all(
                    color: castShadowColor(brightness),
                    width: Cobalt.borderContainer,
                  ),
                  boxShadow: cast(brightness),
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: Cobalt.space3,
                    vertical: Cobalt.space2,
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (final item in DockItem.values)
                        Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: Cobalt.space1,
                          ),
                          child: Builder(
                            builder: (context) {
                              final target = _target(context, item);
                              final enabled =
                                  target.path != null || target.pulse;
                              return _DockButton(
                                item: item,
                                selected: item == current,
                                unfinished:
                                    item == unfinished && item != current,
                                enabled: enabled,
                                iconOnly: narrow,
                                label: l10n.t(_labelKey(item)),
                                unfinishedLabel: l10n.t('dock.unfinished', {
                                  'step': l10n.t(_labelKey(item)),
                                }),
                                onTap: enabled
                                    ? () => _activate(context, item)
                                    : null,
                              );
                            },
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  static String _labelKey(DockItem item) => switch (item) {
    DockItem.capture => 'dock.capture',
    DockItem.ideas => 'dock.ideas',
    DockItem.version => 'dock.version',
    DockItem.commit => 'dock.commit',
    DockItem.roadmap => 'dock.roadmap',
    DockItem.evidence => 'dock.evidence',
  };

  static DockItem? _current(String path) {
    if (path.contains('/ideas')) return DockItem.ideas;
    if (path.contains('/versions')) return DockItem.commit;
    if (path.contains('/roadmap')) return DockItem.roadmap;
    if (path.startsWith('/prd/')) return DockItem.version;
    if (path.startsWith('/projects/')) return DockItem.evidence;
    return null;
  }

  /// First earlier step the loop has not finished. A working version with no
  /// human commit leaves Commit unfinished, ahead of the roadmap and evidence.
  static DockItem? _firstUnfinished(LoopFocus focus) {
    if (focus.prdCommitted == false) return DockItem.commit;
    return null;
  }

  /// The project the current route is already on, so a step with no remembered
  /// id stays here instead of dropping to the dashboard.
  static String? _projectOnRoute(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    final match = RegExp(r'^/projects/([^/]+)').firstMatch(path);
    return match?.group(1);
  }

  /// The workspace the current route is already on, so Capture/Ideas target
  /// the workspace being viewed rather than a stale remembered one.
  static String? _workspaceOnRoute(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    final match = RegExp(r'^/workspace/([^/]+)').firstMatch(path);
    return match?.group(1);
  }

  static _DockTarget _target(BuildContext context, DockItem item) {
    final focus = context.read<LoopFocus>();
    final location = GoRouterState.of(context).uri.path;
    final workspace = _workspaceOnRoute(context) ?? focus.workspaceId;
    final project =
        _projectOnRoute(context) ??
        (workspace != null ? focus.projectFor(workspace) : null) ??
        focus.projectId;
    return switch (item) {
      DockItem.capture || DockItem.ideas => (
        path: workspace != null ? '/workspace/$workspace/ideas' : null,
        pulse: false,
      ),
      // Version rides to the versions route; from the PRD desk the old
      // /prd/<id> target was a self-navigation dead click.
      DockItem.version || DockItem.commit => (
        path: focus.prdId != null ? '/prd/${focus.prdId}/versions' : null,
        pulse: false,
      ),
      DockItem.roadmap => (
        path: project != null ? '/projects/$project/roadmap' : null,
        pulse: false,
      ),
      DockItem.evidence => (
        path: project != null && location != '/projects/$project'
            ? '/projects/$project'
            : null,
        pulse: project != null && location == '/projects/$project',
      ),
    };
  }

  static void _activate(BuildContext context, DockItem item) {
    final (:path, :pulse) = _target(context, item);
    if (pulse) {
      // Already on the evidence destination: no re-push, ask the page to
      // scroll-highlight its evidence section instead.
      context.read<LoopFocus>().pulseEvidence();
      return;
    }
    if (path == null) return; // disabled items never fire
    if (GoRouterState.of(context).uri.path == path) return; // already there
    context.go(path);
  }
}

class _DockButton extends StatelessWidget {
  const _DockButton({
    required this.item,
    required this.selected,
    required this.unfinished,
    required this.enabled,
    required this.iconOnly,
    required this.label,
    required this.unfinishedLabel,
    required this.onTap,
  });

  final DockItem item;
  final bool selected;
  final bool unfinished;
  final bool enabled;

  /// Narrow dock: icon + warn square only; the label moves to the tooltip
  /// and semantics so six items always fit.
  final bool iconOnly;
  final String label;
  final String unfinishedLabel;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Tooltip(
      message: unfinished ? unfinishedLabel : label,
      child: Opacity(
        opacity: enabled ? 1 : 0.5,
        child: PressableBlock(
          key: unfinished ? ValueKey('dock-unfinished-${item.name}') : null,
          onTap: onTap,
          // The dock is brand-filled: a brand ring would vanish on it.
          ringColor: palette.btnFg,
          selected: selected,
          child: AnimatedContainer(
            duration: Cobalt.pressDuration,
            padding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
              vertical: Cobalt.space2,
            ),
            // The 16% wash below is only ~1.4:1 against its neighbours, so
            // the current item also gets a 2px btnFg edge. It is painted over
            // the item, not laid out as a border, which would widen all six
            // and overflow the icon-only dock at 390px.
            foregroundDecoration: selected
                ? BoxDecoration(
                    border: Border.all(
                      color: palette.btnFg,
                      width: Cobalt.borderContainer,
                    ),
                  )
                : null,
            decoration: BoxDecoration(
              color: selected
                  ? Color.alphaBlend(
                      palette.btnFg.withValues(alpha: 0.16),
                      palette.brandHover,
                    )
                  : palette.brandHover,
              borderRadius: BorderRadius.circular(Cobalt.radiusPill),
            ),
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(_icon(item), size: 20, color: palette.btnFg),
                    if (!iconOnly) ...[
                      const SizedBox(height: Cobalt.space1),
                      ConstrainedBox(
                        constraints: const BoxConstraints(maxWidth: 180),
                        child: Text(
                          unfinished ? unfinishedLabel : label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          softWrap: false,
                          style: Theme.of(context).textTheme.labelSmall
                              ?.copyWith(color: palette.btnFg),
                        ),
                      ),
                    ],
                  ],
                ),
                // Amber warn square for unfinished steps — the warning
                // channel, 3.53:1 on the cobalt dock (DESIGN.md).
                if (unfinished)
                  const Positioned(
                    top: 0,
                    right: 0,
                    child: SizedBox(
                      width: Cobalt.space2,
                      height: Cobalt.space2,
                      child: ColoredBox(color: Cobalt.warnAmber),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  static IconData _icon(DockItem item) => switch (item) {
    DockItem.capture => Icons.add,
    DockItem.ideas => Icons.auto_awesome_outlined,
    DockItem.version => Icons.description_outlined,
    DockItem.commit => Icons.verified_outlined,
    DockItem.roadmap => Icons.map_outlined,
    DockItem.evidence => Icons.fact_check_outlined,
  };
}
