import 'package:flutter_test/flutter_test.dart';

import '_roadmap_canvas_helpers.dart';

// The Canvas's layered layout (2026-11-22): columns by dependency, each
// ordered to cross as few links as it can, each box level with what it
// links to.

void main() {
  testWidgets('the Spark flow lines each step up with what it waits on', (
    tester,
  ) async {
    // Onboarding, Matching and Safety start now; Chat waits on Matching;
    // Launch waits on all four.
    await pumpCanvas(tester, [
      canvasNode('R', null),
      canvasNode('O', 'R', position: 0),
      canvasNode('M', 'R', position: 1),
      canvasNode('S', 'R', position: 2),
      canvasNode('C', 'R', position: 3, dependsOn: ['M']),
      canvasNode('L', 'R', position: 4, dependsOn: ['O', 'M', 'S', 'C']),
    ]);
    double mid(String id) => stepCard(tester, id).center.dy;

    expect(mid('C'), moreOrLessEquals(mid('M'), epsilon: 1));
    final feeds = [
      for (final id in ['O', 'M', 'S', 'C']) mid(id),
    ];
    expect(
      mid('L'),
      inInclusiveRange(
        feeds.reduce((a, b) => a < b ? a : b),
        feeds.reduce((a, b) => a > b ? a : b),
      ),
    );
    expect(
      stepCard(tester, 'C').left,
      greaterThan(stepCard(tester, 'M').right),
    );
    expect(
      stepCard(tester, 'L').left,
      greaterThan(stepCard(tester, 'C').right),
    );
  });

  testWidgets('a column is reordered to uncross its links', (tester) async {
    // By position B sits under A, but A feeds the lower of D and C.
    await pumpCanvas(tester, [
      canvasNode('R', null),
      canvasNode('A', 'R', position: 0),
      canvasNode('B', 'R', position: 1),
      canvasNode('C', 'R', position: 2, dependsOn: ['B']),
      canvasNode('D', 'R', position: 3, dependsOn: ['A']),
    ]);
    double mid(String id) => stepCard(tester, id).center.dy;
    expect(mid('A') < mid('B'), mid('D') < mid('C'));
  });
}
