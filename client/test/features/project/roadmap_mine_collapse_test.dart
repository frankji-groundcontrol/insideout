import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEX-03: the roadmap page's Mine view shows the caller's topmost owned
// nodes and their subtrees, nothing else. rv5-TREEX-09: the canvas folds a
// branch and keeps it folded for the page session. rv5-TREEX-10, FE-11: a
// card's bar and count come from the server roll-up, validated included.

Future<void> _view(WidgetTester tester, String label) async {
  await tester.tap(find.text(label));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Mine shows only the caller\'s nodes and their subtrees', (
    tester,
  ) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(
      tester,
      treeSession(api, userId: 'u-dana'),
    );
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _view(tester, 'Mine');

    expect(find.text('Payments'), findsWidgets);
    expect(find.text('Refunds'), findsWidgets);
    expect(find.text('Ship'), findsNothing);
    expect(find.text('Search'), findsNothing);
  });

  testWidgets('Mine with nothing owned says so', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _view(tester, 'Mine');

    expect(find.text('Payments'), findsNothing);
    expect(find.text('Nothing in this tree is yours yet.'), findsOneWidget);
  });

  testWidgets('collapsing a branch hides its descendants for the session', (
    tester,
  ) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _view(tester, 'Canvas');
    expect(find.text('Refunds'), findsOneWidget);

    await tester.tap(find.byKey(const Key('roadmap-collapse-a')));
    await tester.pumpAndSettle();
    expect(find.text('Refunds'), findsNothing);
    expect(find.text('Payments'), findsWidgets);
    expect(find.text('+1'), findsOneWidget);

    // Folding the root hides the whole branch, its own children included.
    await tester.tap(find.byKey(const Key('roadmap-collapse-r')));
    await tester.pumpAndSettle();
    expect(find.text('Payments'), findsNothing);
    expect(find.text('+3'), findsOneWidget);
    await tester.tap(find.byKey(const Key('roadmap-collapse-r')));
    await tester.pumpAndSettle();

    // Another view and back: the fold is remembered.
    await _view(tester, 'Bands');
    await _view(tester, 'Canvas');
    expect(find.text('Refunds'), findsNothing);

    await tester.tap(find.byKey(const Key('roadmap-collapse-a')));
    await tester.pumpAndSettle();
    expect(find.text('Refunds'), findsOneWidget);
  });

  testWidgets('a parent with one validated and one done leaf shows 2 of 2', (
    tester,
  ) async {
    final parent = RoadmapNode(
      id: 'p',
      projectId: 'p1',
      title: 'Parent',
      description: '',
      status: 'in_progress',
      position: 0,
      canWrite: true,
      rollup: const RoadmapRollup(leaves: 2, done: 2, validated: 1),
    );
    await pumpTreeCard(tester, treeSession(TreeApi()), parent, descendants: 2);

    expect(find.text('2 of 2 done · 1 validated'), findsOneWidget);
    final bar = tester.widget<LinearProgressIndicator>(
      find.byType(LinearProgressIndicator),
    );
    expect(bar.value, 1.0);
  });
}
