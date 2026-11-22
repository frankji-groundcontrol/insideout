import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';

import '_roadmap_canvas_helpers.dart';

// 2026-11-22, the owner: "I do not want to see hierarchy on canvas, only
// things in order. Hierarchy is represented when I click into a certain
// step." The Canvas draws one level, the steps, left to right by their
// dependencies lifted to that level; opening a step shows its own steps.

/// R > {A > {A1, A2}, B > B1, C > C1, D > {D1, D2}}: B1 waits on A1 (A→B),
/// C1 waits on B1 (B→C), D waits on A, D1 and D2 wait on B1 (B→D twice), and
/// A2 waits on A1 (inside A). C waits on X, outside every step.
final _flow = [
  canvasNode('R', null),
  canvasNode('A', 'R'),
  canvasNode('A1', 'A'),
  canvasNode('A2', 'A', position: 1, dependsOn: ['A1']),
  canvasNode('B', 'R', position: 1),
  canvasNode('B1', 'B', dependsOn: ['A1']),
  canvasNode('C', 'R', position: 2, dependsOn: ['X', 'gone']),
  canvasNode('C1', 'C', dependsOn: ['B1']),
  canvasNode('D', 'R', position: 3, dependsOn: ['A']),
  canvasNode('D1', 'D', dependsOn: ['B1']),
  canvasNode('D2', 'D', position: 1, dependsOn: ['B1']),
];

void main() {
  testWidgets('one level only: the root\'s children, never their own', (
    tester,
  ) async {
    await pumpCanvas(tester, _flow);
    for (final id in ['A', 'B', 'C', 'D']) {
      expect(stepBox(id), findsOneWidget, reason: '$id is a step');
    }
    for (final id in ['R', 'A1', 'B1', 'D2']) {
      expect(stepBox(id), findsNothing, reason: '$id is not a step');
    }
    expect(find.text('2 items inside'), findsNWidgets(2));
    expect(find.text('1 item inside'), findsNWidgets(2));
    expect(find.textContaining('below'), findsNothing);
  });

  testWidgets('steps run left to right by their lifted dependencies', (
    tester,
  ) async {
    await pumpCanvas(tester, _flow);
    final a = stepCard(tester, 'A');
    final b = stepCard(tester, 'B');
    final c = stepCard(tester, 'C');
    final d = stepCard(tester, 'D');
    // A chain A → B → C, and D merges A and B: C and D share column 2.
    expect(b.left, greaterThan(a.right));
    expect(c.left, greaterThan(b.right));
    expect(d.left, c.left);
    expect(d.top, greaterThan(c.bottom), reason: 'position order');

    // One arrow per pair, counting the links it carries; the link inside A
    // is dropped.
    expect(canvasEdges(tester).toSet(), {
      (from: 'A', to: 'B', count: 1),
      (from: 'B', to: 'C', count: 1),
      (from: 'A', to: 'D', count: 1),
      (from: 'B', to: 'D', count: 2),
    });
  });

  testWidgets('screen readers hear every arrow', (tester) async {
    final semantics = tester.ensureSemantics();
    await pumpCanvas(tester, _flow);
    expect(find.bySemanticsLabel(RegExp('D waits on B')), findsOneWidget);
    semantics.dispose();
  });

  testWidgets('a link out of view is a Waits on line on the step', (
    tester,
  ) async {
    await pumpCanvas(tester, _flow, titles: {'X': 'Licences'});
    expect(
      find.descendant(
        of: stepBox('C'),
        matching: find.text('Waits on: Licences, another item'),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(of: stepBox('D'), matching: find.textContaining('Wait')),
      findsNothing,
      reason: 'an in-view link is an arrow, not a line',
    );
  });

  testWidgets('several roots are the steps', (tester) async {
    await pumpCanvas(tester, [
      canvasNode('R1', null),
      canvasNode('R1a', 'R1'),
      canvasNode('R2', null, position: 1),
    ]);
    expect(stepBox('R1'), findsOneWidget);
    expect(stepBox('R2'), findsOneWidget);
    expect(stepBox('R1a'), findsNothing);
  });

  testWidgets('a step box fits its content', (tester) async {
    final full = RoadmapNode(
      id: 'F',
      projectId: 'p1',
      parentId: 'R',
      title: 'A long title that runs well past one line of the step box',
      description: '',
      status: 'in_progress',
      position: 0,
      deadline: '2026-12-01T00:00:00Z',
      pressure: 'overdue',
      ownerName: 'dana',
      ownerId: 'u1',
      rollup: const RoadmapRollup(leaves: 5, done: 3),
    );
    await pumpCanvas(tester, [
      canvasNode('R', null),
      full,
      canvasNode('F1', 'F'),
      canvasNode('L', 'R', position: 1),
    ], size: const Size(390, 844));
    expect(tester.takeException(), isNull);
    final card = stepCard(tester, 'F');
    for (final text in [
      'Owner: dana',
      '3 of 5 done',
      '1 item inside',
      '2026-12-01 · overdue',
    ]) {
      final line = find.descendant(of: stepBox('F'), matching: find.text(text));
      expect(line, findsOneWidget, reason: text);
      expect(tester.getRect(line).bottom, lessThan(card.bottom), reason: text);
    }
    // A sparse step is short: its block ends just under its title.
    final leaf = stepCard(tester, 'L');
    final title = tester.getRect(
      find.descendant(of: stepBox('L'), matching: find.text('L')),
    );
    expect(leaf.bottom - title.bottom, lessThan(20));
    expect(leaf.height, lessThan(card.height));
  });

  testWidgets('a normal pressure is not shown', (tester) async {
    await pumpCanvas(tester, [
      RoadmapNode(
        id: 'N',
        projectId: 'p1',
        title: 'N',
        description: '',
        status: 'pending',
        position: 0,
        deadline: '2026-12-01T00:00:00Z',
        pressure: 'normal',
      ),
    ]);
    expect(find.text('2026-12-01'), findsOneWidget);
    expect(find.textContaining('normal'), findsNothing);
  });

  testWidgets('the canvas has no fold toggles', (tester) async {
    await pumpCanvas(tester, _flow);
    expect(find.byIcon(Icons.expand_more), findsNothing);
    expect(find.byIcon(Icons.chevron_right), findsNothing);
  });

  testWidgets('a collaborator cursor label stays on the 12px floor', (
    tester,
  ) async {
    await pumpCanvas(
      tester,
      [canvasNode('R', null)],
      cursors: [CursorDot(sessionId: 's1', name: 'Dana', x: 20, y: 20)],
    );
    final style = tester.widget<Text>(find.text('Dana')).style;
    final size =
        style?.fontSize ??
        Theme.of(
          tester.element(find.text('Dana')),
        ).textTheme.bodySmall!.fontSize!;
    expect(size, greaterThanOrEqualTo(12));
  });
}
