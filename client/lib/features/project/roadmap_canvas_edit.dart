part of 'roadmap_canvas.dart';

/// The Canvas's edits: adding a step to the shown layer, and linking two
/// steps by dragging a handle from one onto the other.
extension _CanvasEdit on _RoadmapCanvasState {
  String _t(String key, [Map<String, Object?>? args]) =>
      context.read<Appearance>().t(key, args);

  /// The steps nothing in the layer waits on yet: what a next step waits on.
  List<RoadmapNode> _ends(RoadmapStepFlow flow) => [
    for (final column in flow.columns)
      for (final s in column)
        if (!flow.edges.any((e) => e.from == s.id)) s,
  ];

  /// The "+ Next step" slot: a step that waits on every end of the layer.
  Widget _nextSlot(RoadmapStepFlow flow) => RoadmapNextStepSlot(
    onTap: () =>
        widget.onAddStep!(_t('roadmap.nextStep'), _ends(flow), const []),
  );

  VoidCallback? _addAfter(RoadmapNode s) {
    final add = widget.onAddStep;
    if (add == null) return null;
    return () => add(_t('roadmap.stepAfter', {'title': s.title}), [s], []);
  }

  /// A step beside [s]: waiting on what [s] waits on, waited on by what
  /// waits on [s], so it takes [s]'s column and merges where [s] does.
  VoidCallback? _addParallel(RoadmapNode s, RoadmapStepFlow flow) {
    final add = widget.onAddStep;
    if (add == null) return null;
    final byId = {for (final n in widget.steps) n.id: n};
    final up = [
      for (final e in flow.edges)
        if (e.to == s.id) ?byId[e.from],
    ];
    final down = [
      for (final e in flow.edges)
        if (e.from == s.id) ?byId[e.to],
    ];
    return () => add(_t('roadmap.stepBeside', {'title': s.title}), up, down);
  }

  /// [box] with its link handle on the right edge, when links may be made.
  Widget _linkable(RoadmapNode s, Widget box) {
    if (widget.onLink == null) return box;
    const half = RoadmapLinkHandle.size / 2;
    return Stack(
      children: [
        Padding(
          padding: const EdgeInsets.only(right: half),
          child: box,
        ),
        Positioned(
          right: 0,
          top: 0,
          bottom: 0,
          child: Center(
            child: RoadmapLinkHandle(
              key: Key('roadmap-link-${s.id}'),
              tooltip: _t('roadmap.linkHandle', {'title': s.title}),
              onStart: (g) => _dragTo(s.id, g),
              onMove: (g) => _dragTo(s.id, g),
              onEnd: () => _drop(s),
            ),
          ),
        ),
      ],
    );
  }

  void _dragTo(String from, Offset global) {
    final flow = _flowKey.currentContext?.findRenderObject() as RenderBox?;
    if (flow == null) return;
    _redraw(() => _drag = (from: from, at: flow.globalToLocal(global)));
  }

  /// The drag ends: the step under the pointer, if any other, waits on [s].
  void _drop(RoadmapNode s) {
    final at = _drag?.at;
    _redraw(() => _drag = null);
    if (at == null) return;
    final to = widget.steps
        .where((n) => n.id != s.id && (_rectOf(n.id)?.contains(at) ?? false))
        .firstOrNull;
    if (to != null) widget.onLink?.call(s, to);
  }

  Widget? _dragLine(Color color) {
    final drag = _drag;
    final from = drag == null ? null : _rectOf(drag.from);
    if (drag == null || from == null) return null;
    return Positioned.fill(
      child: IgnorePointer(
        child: CustomPaint(
          key: const Key('roadmap-link-line'),
          painter: RoadmapLinkLinePainter(
            from: from.centerRight,
            to: drag.at,
            color: color,
          ),
        ),
      ),
    );
  }

  /// The steps, their lanes and the next-step slot, placed by the layered
  /// layout (RoadmapFlowLayoutBox): columns by dependency, each ordered to
  /// cross as few links as it can, each box level with what it links to.
  Widget _laidOut(
    RoadmapStepFlow flow,
    ({Map<int, List<String>> byColumn, Map<FlowEdge, List<String>> byEdge})
    lanes,
    String Function(String) title,
  ) {
    const slot = 'roadmap-next-step';
    final add = widget.onAddStep != null;
    final boxes = <String, Widget>{
      for (final column in flow.columns)
        for (final s in column) s.id: _linkable(s, _step(s, flow, title)),
      // Empty room a long link runs through, clear of every box.
      for (final lane in lanes.byColumn.values.expand((l) => l))
        lane: SizedBox(
          key: _keys.putIfAbsent(lane, GlobalKey.new),
          width: RoadmapStepBox.width,
          height: Cobalt.space6,
        ),
      if (add) slot: _nextSlot(flow),
    };
    return RoadmapFlowLayoutBox(
      ids: boxes.keys.toList(),
      columns: [
        for (final (c, column) in flow.columns.indexed)
          [for (final s in column) s.id, ...?lanes.byColumn[c]],
        if (add) [slot],
      ],
      links: [
        ...roadmapFlowLinks(flow.edges, lanes.byEdge),
        if (add)
          for (final end in _ends(flow)) (end.id, slot, 1),
      ],
      groups: flow.loops,
      children: boxes.values.toList(),
    );
  }
}
