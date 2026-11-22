import 'package:flutter/painting.dart';

import '../../theme/cobalt.dart';

typedef FlowEdge = ({String from, String to, int count});

/// The path of each flow arrow, from the laid-out boxes [rectOf] gives.
///
/// A link that skips columns runs through the empty lanes the Canvas reserved
/// for it in each one ([lanes], in column order), so it never passes under a
/// box. Between columns a leg is a curve whose ends point along the flow and
/// stay inside the gap; through a lane it runs straight. Links leaving one
/// step leave from their own points down its right edge, and links into one
/// step land at their own points down its left edge, both ordered by where
/// the link comes from or goes so they do not cross at the box.
Map<FlowEdge, Path> roadmapFlowRoutes(
  List<FlowEdge> edges,
  Map<FlowEdge, List<String>> lanes,
  Rect? Function(String id) rectOf,
) {
  final legs = <FlowEdge, List<Rect>>{};
  for (final e in edges) {
    final rects = [
      rectOf(e.from),
      for (final lane in lanes[e] ?? const <String>[]) rectOf(lane),
      rectOf(e.to),
    ];
    if (rects.every((r) => r != null)) legs[e] = rects.nonNulls.toList();
  }
  final starts = _spread(
    legs,
    (e) => e.from,
    (rects) => rects.first,
    (rects) => rects[1].center.dy,
  );
  final ends = _spread(
    legs,
    (e) => e.to,
    (rects) => rects.last,
    (rects) => rects[rects.length - 2].center.dy,
  );
  return {
    for (final MapEntry(key: e, value: rects) in legs.entries)
      e: _path(rects, starts[e]!, ends[e]!),
  };
}

/// For each box [boxOf] picks, the y its links meet it at: spread evenly
/// down the box, in the order of [toward] (the y they come from or go to).
Map<FlowEdge, double> _spread(
  Map<FlowEdge, List<Rect>> legs,
  String Function(FlowEdge) key,
  Rect Function(List<Rect>) boxOf,
  double Function(List<Rect>) toward,
) {
  final groups = <String, List<FlowEdge>>{};
  for (final e in legs.keys) {
    groups.putIfAbsent(key(e), () => []).add(e);
  }
  final out = <FlowEdge, double>{};
  for (final group in groups.values) {
    group.sort((a, b) => toward(legs[a]!).compareTo(toward(legs[b]!)));
    for (final (i, e) in group.indexed) {
      final box = boxOf(legs[e]!);
      out[e] = box.top + box.height * (i + 1) / (group.length + 1);
    }
  }
  return out;
}

Path _path(List<Rect> rects, double startY, double endY) {
  var at = Offset(rects.first.right, startY);
  final path = Path()..moveTo(at.dx, at.dy);
  void curveTo(Offset to) {
    final pull = (to.dx - at.dx) / 2;
    path.cubicTo(at.dx + pull, at.dy, to.dx - pull, to.dy, to.dx, to.dy);
    at = to;
  }

  for (final lane in rects.sublist(1, rects.length - 1)) {
    curveTo(lane.centerLeft);
    at = lane.centerRight;
    path.lineTo(at.dx, at.dy);
  }
  curveTo(Offset(rects.last.left - Cobalt.borderContainer, endY));
  return path;
}

/// The lanes the Canvas reserves under the boxes of each column a link skips:
/// per column the lane ids to lay out (by where their links start), and per
/// link the lanes it runs through. [columns] holds step ids by column.
({Map<int, List<String>> byColumn, Map<FlowEdge, List<String>> byEdge})
roadmapFlowLanes(List<List<String>> columns, List<FlowEdge> edges) {
  final at = {
    for (final (c, column) in columns.indexed)
      for (final (row, id) in column.indexed) id: (c, row),
  };
  final long =
      [
        for (final e in edges)
          if (at[e.to]!.$1 - at[e.from]!.$1 > 1) e,
      ]..sort((a, b) {
        final (ca, ra) = at[a.from]!;
        final (cb, rb) = at[b.from]!;
        return ca != cb ? ca.compareTo(cb) : ra.compareTo(rb);
      });
  final byColumn = <int, List<String>>{};
  final byEdge = <FlowEdge, List<String>>{};
  for (final e in long) {
    for (var c = at[e.from]!.$1 + 1; c < at[e.to]!.$1; c++) {
      final lane = 'lane:${e.from}>${e.to}@$c';
      byColumn.putIfAbsent(c, () => []).add(lane);
      byEdge.putIfAbsent(e, () => []).add(lane);
    }
  }
  return (byColumn: byColumn, byEdge: byEdge);
}
