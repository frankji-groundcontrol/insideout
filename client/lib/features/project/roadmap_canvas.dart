import 'package:flutter/material.dart';

import '../../api/models.dart';

/// Collaborative canvas v1 (Ink & Seal): each root branch renders as a
/// sibling band — a vertical column of its subtree — laid out
/// horizontally, so parallel efforts read side by side. The minimap is
/// a compact status-tinted overview that jumps to a band.
class RoadmapCanvas extends StatelessWidget {
  const RoadmapCanvas({
    super.key,
    required this.nodes,
    required this.tileBuilder,
    required this.controller,
  });

  final List<RoadmapNode> nodes;
  /// tileBuilder(node, indent) renders one node row (the same editing
  /// affordances as the list view).
  final Widget Function(RoadmapNode node, double indent) tileBuilder;
  final ScrollController controller;

  static const bandWidth = 300.0;
  static const bandSpacing = 16.0;

  List<RoadmapNode> get _roots {
    final roots = nodes.where((n) => n.parentId == null).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
    return roots;
  }

  List<RoadmapNode> _subtree(RoadmapNode root) {
    final out = <RoadmapNode>[root];
    void walk(String parentId, int depth) {
      final kids = nodes.where((n) => n.parentId == parentId).toList()
        ..sort((a, b) => a.position.compareTo(b.position));
      for (final k in kids) {
        out.add(k);
        walk(k.id, depth + 1);
      }
    }

    walk(root.id, 1);
    return out;
  }

  @override
  Widget build(BuildContext context) {
    final roots = _roots;
    return SingleChildScrollView(
      controller: controller,
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.all(16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (var i = 0; i < roots.length; i++) ...[
            if (i > 0) const SizedBox(width: bandSpacing),
            _Band(
              width: bandWidth,
              nodes: _subtree(roots[i]),
              tileBuilder: tileBuilder,
            ),
          ],
        ],
      ),
    );
  }
}

class _Band extends StatelessWidget {
  const _Band({required this.width, required this.nodes, required this.tileBuilder});

  final double width;
  final List<RoadmapNode> nodes;
  final Widget Function(RoadmapNode node, double indent) tileBuilder;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: width,
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: ListView.builder(
        shrinkWrap: true,
        itemCount: nodes.length,
        itemBuilder: (context, i) {
          final node = nodes[i];
          final depth = _depth(node);
          return tileBuilder(node, 8.0 + depth * 18);
        },
      ),
    );
  }

  int _depth(RoadmapNode node) {
    var depth = 0;
    var current = node;
    while (current.parentId != null) {
      final parent = nodes.where((n) => n.id == current.parentId).firstOrNull;
      if (parent == null) break;
      current = parent;
      depth++;
    }
    return depth;
  }
}

/// Minimap: one tiny column per root band, one status-tinted mark per
/// node (three levels deep; deeper nodes fold into their parent's
/// column). Tapping a column scrolls the canvas to that band.
class RoadmapMinimap extends StatelessWidget {
  const RoadmapMinimap({super.key, required this.nodes, required this.onTapBand});

  final List<RoadmapNode> nodes;
  final void Function(int bandIndex) onTapBand;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final roots = nodes.where((n) => n.parentId == null).toList()
      ..sort((a, b) => a.position.compareTo(b.position));

    Color mark(String status) {
      switch (status) {
        case 'done':
          return theme.colorScheme.tertiary;
        case 'in_progress':
          return theme.colorScheme.secondary;
        case 'locked':
          return theme.colorScheme.outline;
        default:
          return theme.colorScheme.outlineVariant;
      }
    }

    return SizedBox(
      height: 84,
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          for (var i = 0; i < roots.length; i++)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 6),
              child: InkWell(
                borderRadius: BorderRadius.circular(4),
                onTap: () => onTapBand(i),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Expanded(
                      child: Container(
                        width: 18,
                        decoration: BoxDecoration(
                          border: Border.all(color: theme.colorScheme.outlineVariant),
                          borderRadius: BorderRadius.circular(3),
                        ),
                        padding: const EdgeInsets.all(2),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                          children: [
                            for (final n in _bandMarks(roots[i]))
                              Container(
                                width: double.infinity,
                                height: 4,
                                decoration: BoxDecoration(
                                  color: mark(n.status),
                                  borderRadius: BorderRadius.circular(1),
                                ),
                              ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      _clip(roots[i].title),
                      style: theme.textTheme.bodySmall?.copyWith(fontSize: 9),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

  /// Up to seven marks per band (root first, then children in order) —
  /// enough shape to recognize a branch at a glance.
  List<RoadmapNode> _bandMarks(RoadmapNode root) {
    final kids = nodes.where((n) => n.parentId == root.id).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
    return [root, ...kids].take(7).toList();
  }

  String _clip(String s) => s.length <= 6 ? s : s.substring(0, 6);
}
