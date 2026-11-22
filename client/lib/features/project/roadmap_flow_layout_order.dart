import 'roadmap_flow_routes.dart';

/// One link between boxes in adjacent columns: from, to, weight.
typedef FlowLink = (String, String, int);

/// Per box, its linked boxes in the column before ([ins]) and after ([outs]),
/// each with the link weight.
typedef FlowNeighbours = ({
  Map<String, List<(String, int)>> ins,
  Map<String, List<(String, int)>> outs,
});

/// The adjacent-column links of [edges]: a link that skips columns runs
/// through its lanes ([byEdge], in column order), one link per leg, each
/// with the edge's count as weight.
List<FlowLink> roadmapFlowLinks(
  List<FlowEdge> edges,
  Map<FlowEdge, List<String>> byEdge,
) => [
  for (final e in edges)
    for (final (a, b) in _legs([e.from, ...?byEdge[e], e.to])) (a, b, e.count),
];

Iterable<(String, String)> _legs(List<String> chain) sync* {
  for (var i = 1; i < chain.length; i++) {
    yield (chain[i - 1], chain[i]);
  }
}

/// The neighbours of every box in [columns] under [links]; a link not between
/// adjacent columns is ignored, and one given right to left is turned round.
FlowNeighbours roadmapFlowNeighbours(
  List<List<String>> columns,
  List<FlowLink> links,
) {
  final column = _columnOf(columns);
  final ins = <String, List<(String, int)>>{};
  final outs = <String, List<(String, int)>>{};
  for (final (a, b, w) in links) {
    final (ca, cb) = (column[a], column[b]);
    if (ca == null || cb == null || (ca - cb).abs() != 1) continue;
    final (from, to) = ca < cb ? (a, b) : (b, a);
    outs.putIfAbsent(from, () => []).add((to, w));
    ins.putIfAbsent(to, () => []).add((from, w));
  }
  return (ins: ins, outs: outs);
}

Map<String, int> _columnOf(List<List<String>> columns) => {
  for (final (c, ids) in columns.indexed)
    for (final id in ids) id: c,
};

/// The weighted count of link pairs that cross between adjacent columns of
/// [order]: two links cross when their ends swap order; the pair counts the
/// product of their weights.
int roadmapFlowCrossings(List<List<String>> order, List<FlowLink> links) {
  final column = _columnOf(order);
  final row = {
    for (final ids in order)
      for (final (i, id) in ids.indexed) id: i,
  };
  final byGap = <int, List<(int, int, int)>>{};
  for (final (a, b, w) in links) {
    final (ca, cb) = (column[a], column[b]);
    if (ca == null || cb == null || (ca - cb).abs() != 1) continue;
    final (from, to) = ca < cb ? (a, b) : (b, a);
    byGap.putIfAbsent(column[from]!, () => []).add((row[from]!, row[to]!, w));
  }
  var crossings = 0;
  for (final legs in byGap.values) {
    for (var i = 0; i < legs.length; i++) {
      for (var j = i + 1; j < legs.length; j++) {
        final (fi, ti, wi) = legs[i];
        final (fj, tj, wj) = legs[j];
        if ((fi - fj) * (ti - tj) < 0) crossings += wi * wj;
      }
    }
  }
  return crossings;
}

/// The order of each column with the fewest crossings found: barycenter
/// sweeps, left to right then back, from [columns]' own order, keeping the
/// best order seen. Ties and boxes with no neighbours on the side being
/// swept keep their place, and each of [groups] stays together as a block.
List<List<String>> roadmapFlowOrder(
  List<List<String>> columns,
  List<FlowLink> links,
  FlowNeighbours neighbours, {
  List<List<String>> groups = const [],
  int sweeps = 8,
}) {
  final groupOf = {
    for (final (g, ids) in groups.indexed)
      for (final id in ids) id: g,
  };
  final order = [
    for (final ids in columns) _reorder(ids, const {}, const {}, groupOf),
  ];
  var best = [for (final ids in order) ids];
  var fewest = roadmapFlowCrossings(order, links);
  for (var s = 0; s < sweeps && fewest > 0; s++) {
    final ahead = s.isEven;
    final n = order.length;
    for (var i = 1; i < n; i++) {
      final c = ahead ? i : n - 1 - i;
      final side = ahead ? neighbours.ins : neighbours.outs;
      final rows = {
        for (final (r, id) in order[ahead ? c - 1 : c + 1].indexed) id: r,
      };
      order[c] = _reorder(order[c], side, rows, groupOf);
    }
    final crossings = roadmapFlowCrossings(order, links);
    if (crossings < fewest) {
      fewest = crossings;
      best = [for (final ids in order) ids];
    }
  }
  return best;
}

/// [column] sorted by barycenter over [side], with each group as one block
/// at its first member's place, its members sorted the same way inside.
List<String> _reorder(
  List<String> column,
  Map<String, List<(String, int)>> side,
  Map<String, int> rows,
  Map<String, int> groupOf,
) {
  final blocks = <List<String>>[];
  final blockOf = <int, List<String>>{};
  for (final id in column) {
    final g = groupOf[id];
    if (g == null) {
      blocks.add([id]);
    } else if (blockOf[g] case final block?) {
      block.add(id);
    } else {
      blocks.add(blockOf[g] = [id]);
    }
  }
  double? bary(List<String> ids) => _barycenter(ids, side, rows);
  return [
    for (final block in _bySlots(blocks, bary))
      if (block.length == 1)
        block.first
      else
        for (final [id] in _bySlots([
          for (final id in block) [id],
        ], bary))
          id,
  ];
}

/// The weighted mean row of the neighbours of [ids], or null with none.
double? _barycenter(
  List<String> ids,
  Map<String, List<(String, int)>> side,
  Map<String, int> rows,
) {
  var sum = 0.0;
  var weight = 0;
  for (final id in ids) {
    for (final (n, w) in side[id] ?? const <(String, int)>[]) {
      final row = rows[n];
      if (row == null) continue;
      sum += row * w;
      weight += w;
    }
  }
  return weight == 0 ? null : sum / weight;
}

/// [items] sorted by [baryOf]; an item without one keeps its slot, so
/// unlinked boxes stay put, and ties keep their order so the sort is stable.
List<T> _bySlots<T>(List<T> items, double? Function(T) baryOf) {
  final bary = [for (final item in items) baryOf(item)];
  final moving =
      [
        for (var i = 0; i < items.length; i++)
          if (bary[i] != null) i,
      ]..sort((a, b) {
        final byBary = bary[a]!.compareTo(bary[b]!);
        return byBary != 0 ? byBary : a.compareTo(b);
      });
  final out = [...items];
  var next = 0;
  for (var i = 0; i < items.length; i++) {
    if (bary[i] != null) out[i] = items[moving[next++]];
  }
  return out;
}
