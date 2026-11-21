import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

import '_roadmap_tree_helpers.dart';

// 2026-11-22 Canvas graph: the Canvas draws the opened root(s) at the top and
// two levels below, joined by connector lines. A deeper subtree reads as a
// "+N" summary, a node opens as the root its owner sees, and a dependency is a
// dashed arrow between drawn nodes or a "Waits on" line for one out of view.

RoadmapNode _n(
  String id,
  String? parent, {
  int position = 0,
  List<String> dependsOn = const [],
}) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: id,
  description: '',
  status: 'pending',
  position: position,
  dependsOn: dependsOn,
);

/// R > {A > A1 > A1x > A1xy, B}, and a second root S.
final _forest = [
  _n('R', null),
  _n('A', 'R'),
  _n('B', 'R', position: 1, dependsOn: ['A1']),
  _n('A1', 'A'),
  _n('A1x', 'A1'),
  _n('A1xy', 'A1x'),
  _n('S', null, position: 1, dependsOn: ['A1x', 'gone']),
];

Future<void> _pump(
  WidgetTester tester,
  List<RoadmapNode> nodes, {
  Size size = const Size(1400, 900),
  List<CursorDot> cursors = const [],
}) async {
  await tester.binding.setSurfaceSize(size);
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
      child: MaterialApp(
        home: Scaffold(
          body: RoadmapCanvas(
            projectId: 'p1',
            nodes: nodes,
            titles: {for (final n in nodes) n.id: n.title},
            cursors: cursors,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Finder _box(String id) => find.byKey(Key('roadmap-graph-node-$id'));

void main() {
  testWidgets('draws the root and two levels below it, no deeper', (
    tester,
  ) async {
    await _pump(tester, _forest);
    for (final id in ['R', 'A', 'B', 'A1', 'S']) {
      expect(_box(id), findsOneWidget, reason: '$id is within two levels');
    }
    expect(_box('A1x'), findsNothing);
    expect(_box('A1xy'), findsNothing);

    // Top-down: each level sits below its parent, siblings side by side.
    final r = tester.getRect(_box('R'));
    final a = tester.getRect(_box('A'));
    final b = tester.getRect(_box('B'));
    final a1 = tester.getRect(_box('A1'));
    expect(a.top, greaterThan(r.bottom));
    expect(a1.top, greaterThan(a.bottom));
    expect(b.top, a.top);
    expect(b.left, greaterThan(a.right));
    // The parent is centred over its children.
    expect(r.center.dx, closeTo((a.center.dx + b.center.dx) / 2, 0.5));
  });

  testWidgets('a deeper subtree reads as a +N summary on its last drawn node', (
    tester,
  ) async {
    await _pump(tester, _forest);
    expect(
      find.descendant(of: _box('A1'), matching: find.text('+2 below')),
      findsOneWidget,
    );
    // Nodes whose children are drawn carry no summary.
    expect(find.textContaining('below'), findsOneWidget);
  });

  testWidgets('several roots sit side by side as sibling trees', (
    tester,
  ) async {
    await _pump(tester, _forest);
    final r = tester.getRect(_box('R'));
    final s = tester.getRect(_box('S'));
    expect(s.top, r.top);
    expect(s.left, greaterThan(tester.getRect(_box('B')).right));
  });

  testWidgets('a dependency between drawn nodes is a dashed arrow', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await _pump(tester, _forest);
    final painter =
        tester
                .widget<CustomPaint>(
                  find.byWidgetPredicate(
                    (w) =>
                        w is CustomPaint &&
                        w.painter is RoadmapDependencyPainter,
                  ),
                )
                .painter!
            as RoadmapDependencyPainter;
    expect(painter.edges, [(from: 'A1', to: 'B')]);
    // Screen readers hear the link the arrow draws.
    expect(find.bySemanticsLabel(RegExp('B waits on A1')), findsOneWidget);
    // An in-view dependency is not repeated as a text line.
    expect(
      find.descendant(of: _box('B'), matching: find.textContaining('Waits on')),
      findsNothing,
    );
    semantics.dispose();
  });

  testWidgets('a dependency out of view is a Waits on line on the node', (
    tester,
  ) async {
    await _pump(tester, _forest);
    expect(
      find.descendant(
        of: _box('S'),
        matching: find.text('Waits on: A1x, another item'),
      ),
      findsOneWidget,
    );
  });

  testWidgets('the canvas has no fold toggles', (tester) async {
    await _pump(tester, _forest);
    expect(find.byIcon(Icons.expand_more), findsNothing);
    expect(find.byIcon(Icons.chevron_right), findsNothing);
    expect(find.byKey(const Key('roadmap-collapse-A')), findsNothing);
  });

  testWidgets('a full node fits its box on a 390px phone, and the canvas '
      'scrolls sideways to its other trees', (tester) async {
    final full = RoadmapNode(
      id: 'F',
      projectId: 'p1',
      parentId: 'Y',
      title: 'A long title that runs well past one line of the node box',
      description: '',
      status: 'in_progress',
      position: 0,
      deadline: '2026-12-01T00:00:00Z',
      pressure: 'overdue',
      ownerId: 'u1',
      ownerName: 'dana',
      dependsOn: const ['elsewhere'],
      rollup: const RoadmapRollup(leaves: 3, done: 1, validated: 1),
    );
    await _pump(tester, [
      _n('X', null),
      _n('Y', 'X'),
      full,
      _n('F1', 'F'),
      _n('F2', 'F1'),
      _n('T', null, position: 1),
    ], size: const Size(390, 844));
    expect(tester.takeException(), isNull);
    final box = _box('F');
    await tester.ensureVisible(box);
    await tester.pumpAndSettle();
    final card = tester.getRect(
      find.descendant(of: box, matching: find.byType(FieldCard)),
    );
    // Every line sits inside the block, the last one included.
    for (final text in [
      'Owner: dana',
      '1 of 3 done · 1 validated',
      'Waits on: another item',
      '2026-12-01 · overdue',
    ]) {
      final line = find.descendant(of: box, matching: find.text(text));
      expect(line, findsOneWidget, reason: text);
      expect(tester.getRect(line).bottom, lessThan(card.bottom), reason: text);
    }
    // The deeper subtree hangs below it, where the next level would start.
    final below = find.descendant(of: box, matching: find.text('+2 below'));
    expect(tester.getRect(below).top, greaterThan(card.bottom));
    final sideways = tester.state<ScrollableState>(
      find.descendant(
        of: find.byType(RoadmapCanvas),
        matching: find.byWidgetPredicate(
          (w) => w is Scrollable && w.axis == Axis.horizontal,
        ),
      ),
    );
    expect(sideways.position.maxScrollExtent, greaterThan(0));
  });

  testWidgets('a normal pressure is not shown', (tester) async {
    await _pump(tester, [
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

  testWidgets('a collaborator cursor label stays on the 12px floor', (
    tester,
  ) async {
    await _pump(
      tester,
      [_n('R', null)],
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

  testWidgets('clicking a node opens it as the root its owner sees', (
    tester,
  ) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Payments'));
    await tester.pumpAndSettle();
    final uri = router.routerDelegate.currentConfiguration.uri;
    expect(uri.path, '/projects/p1/roadmap/a');
    // The opened node is now the root at the top, its child below it, and
    // the page stays on the Canvas.
    expect(uri.queryParameters['view'], 'canvas');
    expect(_box('r'), findsNothing);
    expect(
      tester.getRect(_box('a1')).top,
      greaterThan(tester.getRect(_box('a')).bottom),
    );
  });
}
