import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/features/project/roadmap_step_box.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEB-10 on the web: a subtree with history deletes only after the
// person has read what goes with it, and then with force. rv5-TREEX-02:
// deleting the opened root leaves for the parent's tree, not a dead route.

/// Deletes [title] from its menu in [view]: the Canvas shows steps, so an
/// opened root is deleted from its Bands card.
Future<void> _deleteFromMenu(
  WidgetTester tester,
  String title, {
  String view = 'Canvas',
}) async {
  await tester.tap(find.text(view));
  await tester.pumpAndSettle();
  final card = find
      .ancestor(
        of: find.text(title),
        matching: find.byWidgetPredicate(
          (w) => w is RoadmapNodeCard || w is RoadmapStepBox,
        ),
      )
      .first;
  await tester.tap(
    find.descendant(of: card, matching: find.byType(PopupMenuButton<String>)),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text('Delete').last);
  await tester.pumpAndSettle();
  // The blast-radius confirm comes first.
  await tester.tap(find.text('Delete').last);
  await tester.pumpAndSettle();
}

const _lossLine =
    'Deleting it removes 2 nodes, 1 evidence record, 1 done and 0 validated '
    'items. This cannot be undone.';

void main() {
  testWidgets('a branch with history states the loss and sends force', (
    tester,
  ) async {
    final api = TreeApi()..history = true;
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _deleteFromMenu(tester, 'Payments');

    expect(find.text('This branch has recorded work'), findsOneWidget);
    expect(find.text(_lossLine), findsOneWidget);
    await tester.tap(find.text('Delete anyway'));
    await tester.pumpAndSettle();

    expect(api.deletes, [
      (nodeId: 'a', force: false),
      (nodeId: 'a', force: true),
    ]);
    expect(find.text('Payments'), findsNothing);
  });

  testWidgets('declining the loss sends no forced delete', (tester) async {
    final api = TreeApi()..history = true;
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _deleteFromMenu(tester, 'Payments');

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(api.deletes, [(nodeId: 'a', force: false)]);
    expect(find.text('Payments'), findsWidgets);
  });

  testWidgets('deleting the opened root opens the parent\'s tree', (
    tester,
  ) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap/a');
    await tester.pumpAndSettle();
    await _deleteFromMenu(tester, 'Payments', view: 'Bands');

    expect(
      router.routeInformationProvider.value.uri.path,
      '/projects/p1/roadmap/r',
    );
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();
    expect(find.text('Search'), findsWidgets);
  });

  testWidgets('deleting an opened top-level root opens the whole tree', (
    tester,
  ) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap/r');
    await tester.pumpAndSettle();
    await _deleteFromMenu(tester, 'Ship', view: 'Bands');

    expect(
      router.routeInformationProvider.value.uri.path,
      '/projects/p1/roadmap',
    );
  });
}
