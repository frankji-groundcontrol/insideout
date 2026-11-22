import 'dart:math' as math;

import '../../api/models.dart';
import 'roadmap_tree.dart';

// The Canvas's one level, kept pure in this one file so a server endpoint can
// replace it cleanly: which nodes are the steps, and their order.

/// The steps of [all]: the children of its one root (the whole project, or
/// the opened node), else its roots. A lone root with nothing below it is its
/// own step.
List<RoadmapNode> roadmapSteps(List<RoadmapNode> all) {
  final roots = roadmapRoots(all);
  if (roots.length != 1) return roots;
  final kids = all.where((n) => n.parentId == roots.first.id).toList()
    ..sort((a, b) => a.position.compareTo(b.position));
  return kids.isEmpty ? roots : kids;
}

typedef StepEdge = ({String from, String to, int count});

/// The steps in order. Every dependency in [all] is lifted to the steps that
/// hold its two ends; a link inside one step, or one running up or down one
/// branch, drops. Steps that wait on each other through their work form a
/// loop group. Groups take columns by longest path, so parallel streams run
/// side by side and merge where a step waits on several, and a loop's steps
/// stack in one column.
class RoadmapStepFlow {
  RoadmapStepFlow._(this.columns, this.edges, this.loops, this.waitsOn);

  /// The steps by column; a column holds its groups whole, by position.
  final List<List<RoadmapNode>> columns;

  /// One arrow per pair of steps in different groups, with its link count.
  final List<StepEdge> edges;

  /// The loop groups, two steps or more each, in column order.
  final List<List<String>> loops;

  /// Per step, the ids its work waits on outside every shown step.
  final Map<String, List<String>> waitsOn;

  factory RoadmapStepFlow.of(List<RoadmapNode> steps, List<RoadmapNode> all) {
    final byId = {for (final n in all) n.id: n, for (final s in steps) s.id: s};
    final (counts, waitsOn) = _lift(steps, byId);
    final groups = _groups([for (final s in steps) s.id], counts.keys);
    final groupOf = {
      for (var g = 0; g < groups.length; g++)
        for (final id in groups[g]) id: g,
    };
    final column = _layer(groups.length, [
      for (final (from, to) in counts.keys)
        if (groupOf[from] != groupOf[to]) (groupOf[from]!, groupOf[to]!),
    ]);
    int pos(String id) => byId[id]!.position;
    final order = List.generate(groups.length, (g) => g)
      ..sort((a, b) {
        final byColumn = column[a].compareTo(column[b]);
        if (byColumn != 0) return byColumn;
        return groups[a]
            .map(pos)
            .reduce(math.min)
            .compareTo(groups[b].map(pos).reduce(math.min));
      });
    final columns = <List<RoadmapNode>>[];
    for (final g in order) {
      if (columns.length <= column[g]) columns.add([]);
      columns.last.addAll(
        [for (final id in groups[g]) byId[id]!]
          ..sort((a, b) => a.position.compareTo(b.position)),
      );
    }
    return RoadmapStepFlow._(
      columns,
      [
        for (final MapEntry(key: (from, to), value: count) in counts.entries)
          if (groupOf[from] != groupOf[to]) (from: from, to: to, count: count),
      ],
      [
        for (final g in order)
          if (groups[g].length > 1)
            [
              for (final s in columns.expand((c) => c))
                if (groupOf[s.id] == g) s.id,
            ],
      ],
      waitsOn,
    );
  }
}

/// The links between steps with their counts, and per step the outside ids
/// its work waits on.
(Map<(String, String), int>, Map<String, List<String>>) _lift(
  List<RoadmapNode> steps,
  Map<String, RoadmapNode> byId,
) {
  final ids = {for (final s in steps) s.id};
  Iterable<String> up(String id) sync* {
    for (var n = byId[id]; n != null; n = byId[n.parentId]) {
      yield n.id;
    }
  }

  String? stepOf(String id) => up(id).where(ids.contains).firstOrNull;
  final counts = <(String, String), int>{};
  final waitsOn = <String, List<String>>{};
  for (final n in byId.values) {
    final to = stepOf(n.id);
    if (to == null) continue;
    for (final d in n.dependsOn) {
      // Up or down one branch: the tree already orders the two.
      if (up(n.id).contains(d) || up(d).contains(n.id)) continue;
      final from = stepOf(d);
      if (from == null) {
        final list = waitsOn.putIfAbsent(to, () => []);
        if (!list.contains(d)) list.add(d);
      } else if (from != to) {
        counts.update((from, to), (c) => c + 1, ifAbsent: () => 1);
      }
    }
  }
  return (counts, waitsOn);
}

/// The strongly connected groups of [ids] under [links] (Tarjan): steps that
/// reach each other share a group; every other step is a group of one.
List<List<String>> _groups(List<String> ids, Iterable<(String, String)> links) {
  final next = <String, List<String>>{};
  for (final (from, to) in links) {
    next.putIfAbsent(from, () => []).add(to);
  }
  final index = <String, int>{};
  final low = <String, int>{};
  final stack = <String>[];
  final onStack = <String>{};
  final groups = <List<String>>[];
  void visit(String v) {
    index[v] = low[v] = index.length;
    stack.add(v);
    onStack.add(v);
    for (final w in next[v] ?? const <String>[]) {
      if (!index.containsKey(w)) {
        visit(w);
        low[v] = math.min(low[v]!, low[w]!);
      } else if (onStack.contains(w)) {
        low[v] = math.min(low[v]!, index[w]!);
      }
    }
    if (low[v] != index[v]) return;
    final group = <String>[];
    String w;
    do {
      w = stack.removeLast();
      onStack.remove(w);
      group.add(w);
    } while (w != v);
    groups.add(group);
  }

  for (final id in ids) {
    if (!index.containsKey(id)) visit(id);
  }
  return groups;
}

/// Each of [count] groups' column: one past the furthest group it waits on.
/// The group links are acyclic, so the walk ends.
List<int> _layer(int count, List<(int, int)> links) {
  final into = <int, List<int>>{};
  for (final (from, to) in links) {
    into.putIfAbsent(to, () => []).add(from);
  }
  final column = List<int?>.filled(count, null);
  int place(int g) => column[g] ??= (into[g] ?? const <int>[]).fold<int>(
    0,
    (most, from) => math.max(most, place(from) + 1),
  );
  return [for (var g = 0; g < count; g++) place(g)];
}
