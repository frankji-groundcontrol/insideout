import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEX-03: the roadmap page's Mine view shows the caller's topmost owned
// nodes as the Canvas's steps, nothing else; what is inside them opens one
// click down. rv5-TREEX-10, FE-11: a card's bar and count come from the
// server roll-up, validated included.

Future<void> _view(WidgetTester tester, String label) async {
  await tester.tap(find.text(label));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Mine\'s steps are the caller\'s topmost owned nodes', (
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

    expect(find.byKey(const Key('roadmap-step-a')), findsOneWidget);
    expect(find.text('1 item inside'), findsOneWidget);
    expect(find.text('Refunds'), findsNothing);
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
