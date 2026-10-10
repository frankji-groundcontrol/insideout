import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../theme/cobalt.dart';

export 'roadmap_minimap.dart';
export 'roadmap_painters.dart';

/// Collaborative canvas v1: each root branch renders as a
/// sibling band — a vertical column of its subtree — laid out
/// horizontally, so parallel efforts read side by side. The minimap is
/// a compact status-tinted overview that jumps to a band.
class CursorDot {
  CursorDot({
    required this.sessionId,
    required this.name,
    required this.x,
    required this.y,
  });

  final String sessionId;
  final String name;
  final double x;
  final double y;
}

class RoadmapCanvas extends StatelessWidget {
  const RoadmapCanvas({
    super.key,
    required this.nodes,
    required this.tileBuilder,
    required this.controller,
    this.onCursor,
    this.cursors = const [],
  });

  final List<RoadmapNode> nodes;

  /// tileBuilder(node, indent) renders one node row (the same editing
  /// affordances as the list view).
  final Widget Function(RoadmapNode node, double indent) tileBuilder;
  final ScrollController controller;

  /// onCursor receives pointer moves in canvas content space (nullable
  /// for tests).
  final void Function(double x, double y)? onCursor;

  /// Other sessions' live cursors, rendered as labeled pointers.
  final List<CursorDot> cursors;

  static const bandWidth = 300.0;
  static const bandSpacing = 16.0;

  /// Up/Down scroll the band that holds focus on every platform. Flutter
  /// binds the arrows to scrolling on the web only; elsewhere they move focus
  /// to the neighboring card. PageUp/PageDown already scroll natively.
  static const _arrowScroll = <ShortcutActivator, Intent>{
    SingleActivator(LogicalKeyboardKey.arrowUp): ScrollIntent(
      direction: AxisDirection.up,
    ),
    SingleActivator(LogicalKeyboardKey.arrowDown): ScrollIntent(
      direction: AxisDirection.down,
    ),
  };

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
    final content = Shortcuts(
      shortcuts: _arrowScroll,
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
    return SingleChildScrollView(
      controller: controller,
      scrollDirection: Axis.horizontal,
      padding: EdgeInsets.fromLTRB(
        16,
        16,
        16,
        ProductDock.height + 16 + MediaQuery.of(context).padding.bottom,
      ),
      child: MouseRegion(
        onHover: onCursor == null
            ? null
            : (e) => onCursor!(e.localPosition.dx, e.localPosition.dy),
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            content,
            for (final c in cursors)
              Positioned(
                left: c.x,
                top: c.y,
                child: IgnorePointer(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Icon(
                        Icons.navigation,
                        size: 14,
                        color: Theme.of(context).colorScheme.secondary,
                      ),
                      Text(
                        c.name,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _Band extends StatelessWidget {
  const _Band({
    required this.width,
    required this.nodes,
    required this.tileBuilder,
  });

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
        borderRadius: BorderRadius.circular(Cobalt.radiusCard),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: ListView.builder(
        shrinkWrap: true,
        itemCount: nodes.length,
        itemBuilder: (context, i) {
          final node = nodes[i];
          final depth = _depth(node);
          return Padding(
            padding: EdgeInsets.only(left: depth * 18),
            child: tileBuilder(node, 0),
          );
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
