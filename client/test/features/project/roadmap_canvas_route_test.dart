import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';

import '_roadmap_canvas_helpers.dart';

// Live check on Spark: links that skipped a column were curves that ran under
// the box between and read as ending there. A long link now takes a reserved
// lane through every column it skips, and links arriving at one step land at
// their own points along its left edge.

/// Column 0: A, E. Column 1: B (waits on A), F (waits on E). Column 2: C
/// waits on B, and on A and E across column 1.
final _skips = [
  canvasNode('R', null),
  canvasNode('A', 'R'),
  canvasNode('E', 'R', position: 1),
  canvasNode('B', 'R', position: 2, dependsOn: ['A']),
  canvasNode('F', 'R', position: 3, dependsOn: ['E']),
  canvasNode('C', 'R', position: 4, dependsOn: ['B', 'A', 'E']),
];

RoadmapFlowPainter _painter(WidgetTester tester) =>
    tester
            .widget<CustomPaint>(
              find.byWidgetPredicate(
                (w) => w is CustomPaint && w.painter is RoadmapFlowPainter,
              ),
            )
            .painter!
        as RoadmapFlowPainter;

Iterable<Offset> _samples(Path path) sync* {
  for (final metric in path.computeMetrics()) {
    for (var d = 0.0; d <= metric.length; d += 2) {
      yield metric.getTangentForOffset(d)!.position;
    }
  }
}

void main() {
  testWidgets('a long link never runs under a box it does not join', (
    tester,
  ) async {
    await pumpCanvas(tester, _skips);
    final painter = _painter(tester);
    final routes = painter.routes();
    expect(routes, hasLength(5));
    final ids = ['A', 'B', 'C', 'E', 'F'];
    for (final MapEntry(key: edge, value: path) in routes.entries) {
      for (final id in ids) {
        if (id == edge.from || id == edge.to) continue;
        final box = painter.rectOf(id)!.deflate(1);
        for (final p in _samples(path)) {
          expect(
            box.contains(p),
            isFalse,
            reason: '${edge.from} → ${edge.to} runs under $id at $p',
          );
        }
      }
    }
  });

  testWidgets('links into one step land at distinct points on its left edge', (
    tester,
  ) async {
    await pumpCanvas(tester, _skips);
    final painter = _painter(tester);
    final target = painter.rectOf('C')!;
    final ends = [
      for (final MapEntry(key: edge, value: path) in painter.routes().entries)
        if (edge.to == 'C')
          path
              .computeMetrics()
              .last
              .getTangentForOffset(path.computeMetrics().last.length)!
              .position,
    ];
    expect(ends, hasLength(3));
    expect(ends.map((p) => p.dy.round()).toSet(), hasLength(3));
    for (final p in ends) {
      expect(p.dx, closeTo(target.left, 3));
      expect(p.dy, inExclusiveRange(target.top, target.bottom));
    }
  });
}
