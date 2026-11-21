import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';

import '_roadmap_canvas_helpers.dart';

// Node links never loop, but lifted to steps they can: on the Spark data a
// beta under Launch waits on matching work, and a matching dashboard waits on
// launch metrics. Steps that wait on each other share one column, stacked by
// position, under one two-headed "loop" mark; a link along one branch drops.

List<List<String>> _loops(WidgetTester tester) =>
    (tester
                .widget<CustomPaint>(
                  find.byWidgetPredicate(
                    (w) => w is CustomPaint && w.painter is RoadmapFlowPainter,
                  ),
                )
                .painter!
            as RoadmapFlowPainter)
        .loops;

void main() {
  testWidgets('two steps that wait on each other stack as one loop', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await pumpCanvas(tester, [
      canvasNode('R', null),
      canvasNode('P', 'R'),
      canvasNode('M', 'R', position: 1),
      canvasNode('M1', 'M', dependsOn: ['P']),
      canvasNode('M2', 'M', position: 1, dependsOn: ['L2']),
      canvasNode('L', 'R', position: 2),
      canvasNode('L1', 'L', dependsOn: ['M1']),
      canvasNode('L2', 'L', position: 1),
    ]);
    final p = stepCard(tester, 'P');
    final m = stepCard(tester, 'M');
    final l = stepCard(tester, 'L');
    expect(m.left, greaterThan(p.right));
    expect(l.left, m.left);
    expect(l.top, greaterThan(m.bottom));
    expect(_loops(tester), [
      ['M', 'L'],
    ]);
    expect(canvasEdges(tester), [(from: 'P', to: 'M', count: 1)]);
    expect(
      find.bySemanticsLabel(RegExp('M and L wait on each other')),
      findsOneWidget,
    );
    semantics.dispose();
  });

  testWidgets('a three-step loop shares one column', (tester) async {
    final semantics = tester.ensureSemantics();
    await pumpCanvas(tester, [
      canvasNode('R', null),
      canvasNode('A', 'R', dependsOn: ['C1']),
      canvasNode('B', 'R', position: 1),
      canvasNode('B1', 'B', dependsOn: ['A']),
      canvasNode('C', 'R', position: 2),
      canvasNode('C1', 'C', dependsOn: ['B1']),
      canvasNode('D', 'R', position: 3, dependsOn: ['C']),
    ]);
    final a = stepCard(tester, 'A');
    expect(stepCard(tester, 'B').left, a.left);
    expect(stepCard(tester, 'C').left, a.left);
    expect(stepCard(tester, 'D').left, greaterThan(a.right));
    expect(_loops(tester), [
      ['A', 'B', 'C'],
    ]);
    expect(canvasEdges(tester), [(from: 'C', to: 'D', count: 1)]);
    expect(
      find.bySemanticsLabel(RegExp('A, B and C wait on each other')),
      findsOneWidget,
    );
    semantics.dispose();
  });

  testWidgets('a link up one branch is neither an arrow nor a line', (
    tester,
  ) async {
    await pumpCanvas(tester, [
      canvasNode('R', null),
      canvasNode('A', 'R'),
      canvasNode('A1', 'A', dependsOn: ['R']),
      canvasNode('B', 'R', position: 1, dependsOn: ['A1']),
    ]);
    expect(find.textContaining('Waits on'), findsNothing);
    expect(canvasEdges(tester), [(from: 'A', to: 'B', count: 1)]);
  });
}
