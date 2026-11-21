import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import 'roadmap_graph_layout.dart';
import 'roadmap_graph_node.dart';
import 'roadmap_painters.dart';

export 'roadmap_painters.dart';

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

/// The Canvas: the opened root(s) as a top-down hierarchy two levels deep
/// (RoadmapGraphLayout), connector lines between parent and children, and
/// dependency arrows between drawn nodes. Several roots, as in a project with
/// many goals or the Mine view, stand side by side. It scrolls both ways.
class RoadmapCanvas extends StatefulWidget {
  const RoadmapCanvas({
    super.key,
    required this.projectId,
    required this.nodes,
    this.titles = const {},
    this.inheritedOwner,
    this.menu,
    this.onCursor,
    this.cursors = const [],
  });

  final String projectId;

  /// The nodes to lay out; their roots are the trees' tops.
  final List<RoadmapNode> nodes;

  /// Node titles by id, beyond [nodes] too, to name what a node waits on.
  final Map<String, String> titles;
  final ({String name, String via})? Function(RoadmapNode node)? inheritedOwner;
  final Widget Function(RoadmapNode node)? menu;

  /// onCursor receives pointer moves in graph space, which is the same for
  /// everyone looking at the same tree (nullable for tests).
  final void Function(double x, double y)? onCursor;

  /// Other sessions' live cursors, rendered as labeled pointers.
  final List<CursorDot> cursors;

  @override
  State<RoadmapCanvas> createState() => _RoadmapCanvasState();
}

class _RoadmapCanvasState extends State<RoadmapCanvas> {
  final _down = ScrollController();
  final _across = ScrollController();

  @override
  void dispose() {
    _down.dispose();
    _across.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final graph = _graph(context);
    return LayoutBuilder(
      builder: (context, box) => Scrollbar(
        controller: _down,
        child: SingleChildScrollView(
          controller: _down,
          padding: EdgeInsets.only(
            bottom: ProductDock.height + MediaQuery.paddingOf(context).bottom,
          ),
          child: SingleChildScrollView(
            controller: _across,
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.all(Cobalt.space4),
            // A graph narrower than the page sits in its middle.
            child: ConstrainedBox(
              constraints: BoxConstraints(
                minWidth: box.maxWidth - 2 * Cobalt.space4,
              ),
              child: Align(alignment: Alignment.topCenter, child: graph),
            ),
          ),
        ),
      ),
    );
  }

  Widget _graph(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final palette = Cobalt.of(context);
    final nodes = widget.nodes;
    final layout = RoadmapGraphLayout.of(nodes);
    final byId = {for (final n in nodes) n.id: n};
    final parents = {for (final n in nodes) n.parentId};
    final drawn = [for (final id in layout.boxes.keys) byId[id]!];
    final edges = [
      for (final n in drawn)
        for (final d in n.dependsOn)
          if (layout.boxes.containsKey(d)) (from: d, to: n.id),
    ];
    String title(String id) => widget.titles[id] ?? t('roadmap.someItem');
    final graph = SizedBox.fromSize(
      size: layout.size,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned.fill(
            child: CustomPaint(
              painter: RoadmapTreePainter(layout, palette.strokeStrong),
            ),
          ),
          for (final n in drawn)
            Positioned(
              left: layout.boxes[n.id]!.left,
              top: layout.boxes[n.id]!.top,
              width: RoadmapGraphLayout.boxWidth,
              child: RoadmapGraphNode(
                key: Key('roadmap-graph-node-${n.id}'),
                projectId: widget.projectId,
                node: n,
                branch: parents.contains(n.id),
                hidden: layout.hidden[n.id] ?? 0,
                waitsOn: [
                  for (final d in n.dependsOn)
                    if (!layout.boxes.containsKey(d)) title(d),
                ],
                inheritedOwner: widget.inheritedOwner?.call(n),
                menu: widget.menu?.call(n),
              ),
            ),
          Positioned.fill(
            child: Semantics(
              label: [
                for (final e in edges)
                  t('roadmap.dependencyEdge', {
                    'to': title(e.to),
                    'from': title(e.from),
                  }),
              ].join('. '),
              child: IgnorePointer(
                child: CustomPaint(
                  painter: RoadmapDependencyPainter(
                    layout,
                    edges,
                    palette.brand,
                  ),
                ),
              ),
            ),
          ),
          for (final c in widget.cursors) _cursor(context, c),
        ],
      ),
    );
    final onCursor = widget.onCursor;
    if (onCursor == null) return graph;
    return MouseRegion(
      onHover: (e) => onCursor(e.localPosition.dx, e.localPosition.dy),
      child: graph,
    );
  }

  Widget _cursor(BuildContext context, CursorDot c) {
    final theme = Theme.of(context);
    return Positioned(
      left: c.x,
      top: c.y,
      child: IgnorePointer(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              Icons.navigation,
              size: 14,
              color: theme.colorScheme.secondary,
            ),
            Text(c.name, style: theme.textTheme.bodySmall),
          ],
        ),
      ),
    );
  }
}
