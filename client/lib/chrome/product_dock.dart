import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../session/appearance.dart';
import '../theme/cobalt.dart';
import 'loop_focus.dart';

enum DockItem { capture, ideas, version, commit, roadmap, evidence }

/// Lock chrome E — floating pill dock. No left rail at any breakpoint.
class ProductDock extends StatelessWidget {
  const ProductDock({super.key});

  static const height = 88.0;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final location = GoRouterState.of(context).uri.path;
    final current = _current(location);
    final palette = Cobalt.of(context);

    return SafeArea(
      minimum: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      child: Center(
        child: Material(
          color: palette.brand,
          elevation: 8,
          shadowColor: Cobalt.lightInk.withValues(alpha: 0.18),
          borderRadius: BorderRadius.circular(Cobalt.radiusPill),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                for (final item in DockItem.values)
                  _DockButton(
                    item: item,
                    selected: item == current,
                    label: l10n.t(_labelKey(item)),
                    onTap: () => _go(context, item),
                  ),
              ],
            ),
          ),
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

  static void _go(BuildContext context, DockItem item) {
    final focus = context.read<LoopFocus>();
    switch (item) {
      case DockItem.capture:
      case DockItem.ideas:
        final ws = focus.workspaceId;
        if (ws != null) {
          context.go('/workspace/$ws/ideas');
        } else {
          context.go('/dashboard');
        }
      case DockItem.version:
        final prd = focus.prdId;
        if (prd != null) {
          context.go('/prd/$prd');
        } else {
          context.go('/dashboard');
        }
      case DockItem.commit:
        final prd = focus.prdId;
        if (prd != null) {
          context.go('/prd/$prd/versions');
        } else {
          context.go('/dashboard');
        }
      case DockItem.roadmap:
        final project = focus.projectId;
        if (project != null) {
          context.go('/projects/$project/roadmap');
        } else {
          context.go('/dashboard');
        }
      case DockItem.evidence:
        final project = focus.projectId;
        if (project != null) {
          context.go('/projects/$project');
        } else {
          context.go('/dashboard');
        }
    }
  }
}

class _DockButton extends StatelessWidget {
  const _DockButton({
    required this.item,
    required this.selected,
    required this.label,
    required this.onTap,
  });

  final DockItem item;
  final bool selected;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Tooltip(
      message: label,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            color: selected ? palette.btnFg.withValues(alpha: 0.16) : Colors.transparent,
            borderRadius: BorderRadius.circular(Cobalt.radiusPill),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(_icon(item), size: 20, color: palette.btnFg),
              const SizedBox(height: 2),
              Text(
                label,
                style: Theme.of(context).textTheme.labelSmall?.copyWith(color: palette.btnFg, fontSize: 11),
              ),
            ],
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
