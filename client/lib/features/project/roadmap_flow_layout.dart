import 'dart:math' as math;

import 'package:flutter/painting.dart';

import 'roadmap_flow_layout_order.dart';

export 'roadmap_flow_layout_order.dart'
    show FlowLink, roadmapFlowCrossings, roadmapFlowLinks;

/// Where each box sits, the size of the whole flow, and the column order
/// chosen.
typedef FlowLayout = ({
  Map<String, Offset> topLeft,
  Size size,
  List<List<String>> order,
});

/// The flow laid out: each column's order chosen to cross as few [links] as
/// it can, then each box moved toward the boxes it links to, so a box lines
/// up with what it connects to.
///
/// [columns] lists every id by column, steps and lanes alike; its order breaks
/// ties. [links] join boxes in adjacent columns only (see [roadmapFlowLinks]).
/// [sizes] holds the measured size of every id. Each of [groups] (a loop)
/// stays together in its column. Columns sit [columnGap] apart, left-aligned
/// at the width of their widest box; boxes in a column at least [rowGap]
/// apart. The flow's top-left box edge is at the origin.
FlowLayout roadmapFlowLayout({
  required List<List<String>> columns,
  required List<FlowLink> links,
  required Map<String, Size> sizes,
  List<List<String>> groups = const [],
  double columnGap = 72,
  double rowGap = 24,
}) {
  final neighbours = roadmapFlowNeighbours(columns, links);
  final order = roadmapFlowOrder(columns, links, neighbours, groups: groups);
  final centre = _centres(order, neighbours, sizes, rowGap);
  final top = centre.map((id, y) => MapEntry(id, y - sizes[id]!.height / 2));
  final shift = top.values.fold(double.infinity, math.min);
  final topLeft = <String, Offset>{};
  var x = 0.0;
  var bottom = 0.0;
  for (final (c, ids) in order.indexed) {
    final width = ids.map((id) => sizes[id]!.width).fold(0.0, math.max);
    for (final id in ids) {
      topLeft[id] = Offset(x, top[id]! - shift);
      bottom = math.max(bottom, topLeft[id]!.dy + sizes[id]!.height);
    }
    x += width + (c == order.length - 1 ? 0 : columnGap);
  }
  return (topLeft: topLeft, size: Size(x, bottom), order: order);
}

/// The centre y of every box: packed, then swept left to right and back,
/// each column settling toward its neighbours on the side the sweep comes
/// from. It ends on a left-to-right sweep, so a box lines up with what it
/// waits on.
Map<String, double> _centres(
  List<List<String>> order,
  FlowNeighbours neighbours,
  Map<String, Size> sizes,
  double rowGap,
) {
  final y = <String, double>{};
  for (final ids in order) {
    var top = 0.0;
    for (final id in ids) {
      final h = sizes[id]!.height;
      y[id] = top + h / 2;
      top += h + rowGap;
    }
  }
  final n = order.length;
  for (var p = 0; p < 11; p++) {
    final ahead = p.isEven;
    for (var i = 1; i < n; i++) {
      final c = ahead ? i : n - 1 - i;
      final side = ahead ? neighbours.ins : neighbours.outs;
      _settle(order[c], side, y, sizes, rowGap);
    }
  }
  return y;
}

/// Moves each box of [ids] toward the weighted mean centre of its neighbours
/// on [side], keeping the order and at least [rowGap] between boxes.
void _settle(
  List<String> ids,
  Map<String, List<(String, int)>> side,
  Map<String, double> y,
  Map<String, Size> sizes,
  double rowGap,
) {
  final want = <double>[];
  final pull = <int>[];
  for (final id in ids) {
    var sum = 0.0;
    var weight = 0;
    for (final (n, w) in side[id] ?? const <(String, int)>[]) {
      sum += y[n]! * w;
      weight += w;
    }
    want.add(weight == 0 ? y[id]! : sum / weight);
    pull.add(weight);
  }
  double height(int i) => sizes[ids[i]]!.height;
  // Linked boxes before unlinked ones, then taller before shorter: a step
  // takes its place and the thin lanes beside it give way.
  final byPriority = List.generate(ids.length, (i) => i)
    ..sort((a, b) {
      int key(int Function(int) f) => f(b).compareTo(f(a));
      final linked = key((i) => pull[i] > 0 ? 1 : 0);
      if (linked != 0) return linked;
      final tall = height(b).compareTo(height(a));
      if (tall != 0) return tall;
      final heavy = key((i) => pull[i]);
      return heavy != 0 ? heavy : a.compareTo(b);
    });
  final at = [for (final id in ids) y[id]!];
  _place(at, want, [
    for (var i = 1; i < ids.length; i++)
      (height(i - 1) + height(i)) / 2 + rowGap,
  ], byPriority);
  for (final (i, id) in ids.indexed) {
    y[id] = at[i];
  }
}

/// Places the centres [at] one by one in [byPriority] order, each as close
/// to [want] as the boxes already placed allow, pushing the rest aside so
/// neighbours i and i+1 stay at least [gap][i] apart.
void _place(
  List<double> at,
  List<double> want,
  List<double> gap,
  List<int> byPriority,
) {
  final placed = List.filled(at.length, false);
  for (final k in byPriority) {
    var lo = double.negativeInfinity;
    var hi = double.infinity;
    for (var i = k - 1, room = 0.0; i >= 0; i--) {
      room += gap[i];
      if (placed[i]) {
        lo = at[i] + room;
        break;
      }
    }
    for (var i = k + 1, room = 0.0; i < at.length; i++) {
      room += gap[i - 1];
      if (placed[i]) {
        hi = at[i] - room;
        break;
      }
    }
    at[k] = math.min(math.max(want[k], lo), hi);
    placed[k] = true;
    for (var i = k + 1; i < at.length && at[i] < at[i - 1] + gap[i - 1]; i++) {
      at[i] = at[i - 1] + gap[i - 1];
    }
    for (var i = k - 1; i >= 0 && at[i] > at[i + 1] - gap[i]; i--) {
      at[i] = at[i + 1] - gap[i];
    }
  }
}
