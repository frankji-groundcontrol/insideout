import 'dart:math' as math;
import 'dart:ui';

import '../../api/models.dart';
import 'roadmap_tree.dart';

/// Where the Canvas draws each node: every root at the top, its children side
/// by side one level below, theirs below that, and nothing deeper. A subtree
/// is as wide as its children together (or one box), its children are centred
/// under it, and the parent sits centred over its first and last child. The
/// roots' trees stand side by side.
class RoadmapGraphLayout {
  RoadmapGraphLayout._(this.boxes, this.links, this.hidden, this.size);

  static const boxWidth = 248.0;
  static const boxHeight = 212.0;
  static const siblingGap = 24.0;
  static const levelGap = 56.0;
  static const treeGap = 48.0;

  /// Room under the last level for a "+N" stub.
  static const stubHeight = 56.0;

  /// Levels drawn below a root.
  static const depth = 2;

  /// Each drawn node's box in canvas space, in drawing order.
  final Map<String, Rect> boxes;

  /// Each drawn parent with its drawn children, in order: the connectors.
  final List<({String parent, List<String> children})> links;

  /// Nodes on the last drawn level with work below them: how many nodes that
  /// subtree holds (the "+N").
  final Map<String, int> hidden;
  final Size size;

  factory RoadmapGraphLayout.of(List<RoadmapNode> nodes) {
    final kids = <String, List<RoadmapNode>>{};
    for (final n in nodes) {
      if (n.parentId != null) kids.putIfAbsent(n.parentId!, () => []).add(n);
    }
    for (final list in kids.values) {
      list.sort((a, b) => a.position.compareTo(b.position));
    }
    final boxes = <String, Rect>{};
    final links = <({String parent, List<String> children})>[];
    final hidden = <String, int>{};
    var levels = 0;

    List<RoadmapNode> drawn(RoadmapNode n, int level) =>
        level < depth ? kids[n.id] ?? const [] : const [];
    double span(List<RoadmapNode> row, double Function(RoadmapNode) w) =>
        row.fold(0.0, (sum, c) => sum + w(c)) + siblingGap * (row.length - 1);
    double width(RoadmapNode n, int level) {
      final row = drawn(n, level);
      if (row.isEmpty) return boxWidth;
      return math.max(boxWidth, span(row, (c) => width(c, level + 1)));
    }

    int below(String id) =>
        (kids[id] ?? const []).fold(0, (sum, c) => sum + 1 + below(c.id));

    void place(RoadmapNode n, int level, double left) {
      levels = math.max(levels, level);
      final top = level * (boxHeight + levelGap);
      final row = drawn(n, level);
      if (row.isEmpty) {
        final w = width(n, level);
        boxes[n.id] = Rect.fromLTWH(
          left + (w - boxWidth) / 2,
          top,
          boxWidth,
          boxHeight,
        );
        if (kids.containsKey(n.id)) hidden[n.id] = below(n.id);
        return;
      }
      final w = width(n, level);
      var x = left + (w - span(row, (c) => width(c, level + 1))) / 2;
      for (final c in row) {
        place(c, level + 1, x);
        x += width(c, level + 1) + siblingGap;
      }
      final mid =
          (boxes[row.first.id]!.center.dx + boxes[row.last.id]!.center.dx) / 2;
      boxes[n.id] = Rect.fromLTWH(mid - boxWidth / 2, top, boxWidth, boxHeight);
      links.add((parent: n.id, children: [for (final c in row) c.id]));
    }

    var left = 0.0;
    for (final root in roadmapRoots(nodes)) {
      if (left > 0) left += treeGap;
      place(root, 0, left);
      left += width(root, 0);
    }
    final height =
        (levels + 1) * boxHeight +
        levels * levelGap +
        (hidden.isEmpty ? 0 : stubHeight);
    return RoadmapGraphLayout._(boxes, links, hidden, Size(left, height));
  }
}
