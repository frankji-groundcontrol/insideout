import 'package:flutter/material.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../theme/cobalt.dart';
import '../../widgets/pressable_block.dart';

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
    final content = Row(
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
                        style: Theme.of(
                          context,
                        ).textTheme.bodySmall?.copyWith(fontSize: 10),
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

/// Minimap: one tiny column per root band, one shape-coded mark per
/// node (three levels deep; deeper nodes fold into their parent's
/// column). Tapping a column scrolls the canvas to that band.
///
/// Shape — not color — carries the state so the marks read in
/// grayscale and for color-blind users: done = filled block,
/// in_progress = half-filled, locked = dashed outline, pending =
/// hollow outline. Every mark also exposes its state to screen
/// readers and hover tooltips via [_MinimapMark].
class RoadmapMinimap extends StatelessWidget {
  const RoadmapMinimap({
    super.key,
    required this.nodes,
    required this.onTapBand,
  });

  final List<RoadmapNode> nodes;
  final void Function(int bandIndex) onTapBand;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final roots = nodes.where((n) => n.parentId == null).toList()
      ..sort((a, b) => a.position.compareTo(b.position));

    return SizedBox(
      height: 84,
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          for (var i = 0; i < roots.length; i++)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 6),
              // The whole column is the pressable block (D7): a decorated
              // container, so every pixel of the band is tappable — no
              // Material ripple in the block language.
              child: PressableBlock(
                onTap: () => onTapBand(i),
                child: Container(
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerHighest,
                    border: Border.all(color: theme.colorScheme.outlineVariant),
                  ),
                  padding: const EdgeInsets.fromLTRB(6, 2, 6, 0),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Expanded(
                        child: SizedBox(
                          width: 18,
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                            children: [
                              for (final n in _bandMarks(roots[i]))
                                _MinimapMark(status: n.status),
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

/// One minimap mark. The shape encodes the node state (see
/// [RoadmapMinimap]); [Semantics]/[Tooltip] carry the same state as
/// text. Labels are plain English — this file has no l10n keys yet;
/// when it gains them, mirror these under `roadmap.minimap.status.*`.
/// Stroke/fill colors come from Cobalt tokens so every state stays
/// ≥3:1 against the sunken column background in both themes.
class _MinimapMark extends StatelessWidget {
  const _MinimapMark({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final cobalt = Cobalt.of(context);
    final Widget mark;
    switch (status) {
      case 'done':
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            color: cobalt.statusSuccessFg,
            borderRadius: BorderRadius.zero,
          ),
        );
      case 'in_progress':
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            border: Border.all(
              color: cobalt.strokeStrong,
              width: Cobalt.borderDivider,
            ),
            borderRadius: BorderRadius.zero,
          ),
          child: FractionallySizedBox(
            alignment: Alignment.centerLeft,
            widthFactor: 0.5,
            child: Container(color: cobalt.fgBrand),
          ),
        );
      case 'locked':
        mark = CustomPaint(
          painter: MinimapDashMarkPainter(cobalt.strokeStrong),
          child: const SizedBox(width: double.infinity, height: 4),
        );
      default:
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            border: Border.all(
              color: cobalt.fgMuted,
              width: Cobalt.borderDivider,
            ),
            borderRadius: BorderRadius.zero,
          ),
        );
    }
    return Semantics(
      container: true,
      label: _label(status),
      child: Tooltip(
        message: _label(status),
        excludeFromSemantics: true,
        child: mark,
      ),
    );
  }

  static String _label(String status) {
    switch (status) {
      case 'done':
        return 'done';
      case 'in_progress':
        return 'in progress';
      case 'locked':
        return 'locked';
      default:
        return 'pending';
    }
  }
}

/// Dashed square outline for locked minimap marks. Stroke-only paint;
/// the dash is drawn as short segments because BoxDecoration has no
/// dashed border.
class MinimapDashMarkPainter extends CustomPainter {
  MinimapDashMarkPainter(this.color);

  final Color color;

  static const _dash = 2.0;
  static const _gap = 1.0;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderDivider;
    // Inset by half the stroke so the outline is not clipped at the edges.
    final rect = Rect.fromLTWH(
      Cobalt.borderDivider / 2,
      Cobalt.borderDivider / 2,
      size.width - Cobalt.borderDivider,
      size.height - Cobalt.borderDivider,
    );
    var x = rect.left;
    while (x < rect.right) {
      final end = (x + _dash).clamp(rect.left, rect.right);
      canvas.drawLine(Offset(x, rect.top), Offset(end, rect.top), paint);
      canvas.drawLine(Offset(x, rect.bottom), Offset(end, rect.bottom), paint);
      x += _dash + _gap;
    }
    var y = rect.top;
    while (y < rect.bottom) {
      final end = (y + _dash).clamp(rect.top, rect.bottom);
      canvas.drawLine(Offset(rect.left, y), Offset(rect.left, end), paint);
      canvas.drawLine(Offset(rect.right, y), Offset(rect.right, end), paint);
      y += _dash + _gap;
    }
  }

  @override
  bool shouldRepaint(MinimapDashMarkPainter oldDelegate) =>
      oldDelegate.color != color;
}
