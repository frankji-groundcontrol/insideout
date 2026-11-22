import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import 'roadmap_canvas_view.dart';
import 'roadmap_flow_routes.dart';
import 'roadmap_painters.dart';
import 'roadmap_step_box.dart';
import 'roadmap_step_flow.dart';
import 'roadmap_tree.dart';

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

/// The Canvas: one level of steps, in order. Steps run left to right in
/// columns by their dependencies (RoadmapStepFlow), arrows join a step to
/// the ones waiting on it, and opening a step shows its own steps. The view
/// pans by drag and zooms by wheel, pinch or the zoom block.
class RoadmapCanvas extends StatefulWidget {
  const RoadmapCanvas({
    super.key,
    required this.projectId,
    required this.steps,
    required this.all,
    this.titles = const {},
    this.inheritedOwner,
    this.menu,
    this.onCursor,
    this.cursors = const [],
    this.docked = true,
  });

  /// The product dock floats over the foot of the view; a node's window has
  /// none.
  final bool docked;

  final String projectId;

  /// The steps to draw (roadmapSteps, or Mine's topmost owned nodes).
  final List<RoadmapNode> steps;

  /// Every loaded node: the dependencies to lift and the items to count.
  final List<RoadmapNode> all;

  /// Node titles by id, beyond [all] too, to name what a step waits on.
  final Map<String, String> titles;
  final ({String name, String via})? Function(RoadmapNode node)? inheritedOwner;
  final Widget Function(RoadmapNode node)? menu;

  /// onCursor receives pointer moves in flow space: the view's pan and zoom
  /// do not change it (nullable for tests).
  final void Function(double x, double y)? onCursor;

  /// Other sessions' live cursors, rendered as labeled pointers.
  final List<CursorDot> cursors;

  @override
  State<RoadmapCanvas> createState() => _RoadmapCanvasState();
}

class _RoadmapCanvasState extends State<RoadmapCanvas> {
  final _view = TransformationController();
  final _flowKey = GlobalKey();
  final _keys = <String, GlobalKey>{};
  var _viewport = Size.zero;
  var _placed = false;

  @override
  void dispose() {
    _view.dispose();
    super.dispose();
  }

  Size? get _flowSize =>
      (_flowKey.currentContext?.findRenderObject() as RenderBox?)?.size;

  /// Where step [id] sits in flow space, once laid out.
  Rect? _rectOf(String id) {
    final box = _keys[id]?.currentContext?.findRenderObject() as RenderBox?;
    final flow = _flowKey.currentContext?.findRenderObject() as RenderBox?;
    if (box == null || flow == null || !box.hasSize) return null;
    return box.localToGlobal(Offset.zero, ancestor: flow) & box.size;
  }

  void _place() {
    final size = _flowSize;
    if (!mounted || _placed || size == null) return;
    _placed = true;
    _view.value = roadmapStartMatrix(size, _viewport);
  }

  void _fit() {
    final size = _flowSize;
    if (size != null) _view.value = roadmapFitMatrix(size, _viewport);
  }

  void _zoom(double factor) => _view.value = roadmapZoomAbout(
    _view.value,
    factor,
    _viewport.center(Offset.zero),
  );

  @override
  Widget build(BuildContext context) {
    final flow = _flow(context);
    if (!_placed) WidgetsBinding.instance.addPostFrameCallback((_) => _place());
    return LayoutBuilder(
      builder: (context, box) {
        // The dock floats over the foot of the view.
        final dock = widget.docked
            ? ProductDock.height + MediaQuery.paddingOf(context).bottom
            : 0.0;
        _viewport = Size(box.maxWidth, box.maxHeight - dock);
        return Stack(
          children: [
            Positioned.fill(
              child: InteractiveViewer(
                transformationController: _view,
                constrained: false,
                boundaryMargin: const EdgeInsets.all(double.infinity),
                minScale: roadmapMinScale,
                maxScale: roadmapMaxScale,
                child: flow,
              ),
            ),
            Positioned(
              top: Cobalt.space3,
              right: Cobalt.space3,
              child: RoadmapZoomControls(
                onZoomIn: () => _zoom(1.25),
                onZoomOut: () => _zoom(0.8),
                onFit: _fit,
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _flow(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final flow = RoadmapStepFlow.of(widget.steps, widget.all);
    String title(String id) => widget.titles[id] ?? t('roadmap.someItem');
    final lanes = roadmapFlowLanes([
      for (final column in flow.columns) [for (final s in column) s.id],
    ], flow.edges);
    final steps = Row(
      spacing: 72,
      children: [
        for (final (c, column) in flow.columns.indexed)
          Column(
            mainAxisSize: MainAxisSize.min,
            spacing: Cobalt.space6,
            children: [
              for (final s in column) _step(s, flow, title),
              // Empty room a long link runs through, clear of every box.
              for (final lane in lanes.byColumn[c] ?? const <String>[])
                SizedBox(
                  key: _keys.putIfAbsent(lane, GlobalKey.new),
                  width: RoadmapStepBox.width,
                  height: Cobalt.space6,
                ),
            ],
          ),
      ],
    );
    final stack = Stack(
      key: _flowKey,
      clipBehavior: Clip.none,
      children: [
        Positioned.fill(
          child: Semantics(
            label: [
              for (final e in flow.edges)
                t('roadmap.dependencyEdge', {
                  'to': title(e.to),
                  'from': title(e.from),
                }),
              for (final group in flow.loops)
                t('roadmap.dependencyLoop', {
                  'steps': _list(t, [for (final id in group) title(id)]),
                }),
            ].join('. '),
            child: CustomPaint(
              painter: RoadmapFlowPainter(
                edges: flow.edges,
                lanes: lanes.byEdge,
                loops: flow.loops,
                loopLabel: t('roadmap.loop'),
                rectOf: _rectOf,
                color: palette.brand,
                labelStyle: theme.textTheme.labelSmall!.copyWith(
                  color: palette.fgBrand,
                ),
                labelFill: palette.surfaceRaised,
              ),
            ),
          ),
        ),
        Padding(padding: const EdgeInsets.all(Cobalt.space6), child: steps),
        for (final c in widget.cursors) _cursor(context, c),
      ],
    );
    final onCursor = widget.onCursor;
    if (onCursor == null) return stack;
    return MouseRegion(
      onHover: (e) => onCursor(e.localPosition.dx, e.localPosition.dy),
      child: stack,
    );
  }

  /// "A, B and C" in the active language.
  static String _list(
    String Function(String, [Map<String, Object?>?]) t,
    List<String> items,
  ) => items.length < 2
      ? items.join()
      : t('roadmap.listAnd', {
          'rest': items.sublist(0, items.length - 1).join(t('roadmap.listSep')),
          'last': items.last,
        });

  Widget _step(
    RoadmapNode s,
    RoadmapStepFlow flow,
    String Function(String) title,
  ) => KeyedSubtree(
    key: _keys.putIfAbsent(s.id, GlobalKey.new),
    child: RoadmapStepBox(
      key: Key('roadmap-step-${s.id}'),
      projectId: widget.projectId,
      node: s,
      inside: roadmapDescendantCount(widget.all, s),
      waitsOn: [for (final d in flow.waitsOn[s.id] ?? const []) title(d)],
      inheritedOwner: widget.inheritedOwner?.call(s),
      menu: widget.menu?.call(s),
    ),
  );

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
