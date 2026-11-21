import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/router.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEX-02, FE-03, TREEX-11: a node opens as its own tree at
// /projects/:id/roadmap/:nodeId, with a breadcrumb back to the whole tree.

Finder _crumb(String text) => find.descendant(
  of: find.byKey(const Key('roadmap-breadcrumb')),
  matching: find.text(text),
);

Future<void> _canvas(WidgetTester tester) async {
  await tester.tap(find.text('Canvas'));
  await tester.pumpAndSettle();
}

void main() {
  test('the node route is registered', () {
    expect(
      productRouteBuilders.keys,
      contains('/projects/:id/roadmap/:nodeId'),
    );
  });

  testWidgets('opening a node shows only its subtree and the breadcrumb', (
    tester,
  ) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap/a');
    await tester.pumpAndSettle();
    await _canvas(tester);

    expect(api.subtreeRoots, contains('a'));
    expect(find.text('Payments'), findsWidgets);
    expect(find.text('Refunds'), findsWidgets);
    expect(find.text('Search'), findsNothing);
    expect(_crumb('Loop project'), findsOneWidget);
    expect(_crumb('Ship'), findsOneWidget);

    // The project crumb goes back to the whole tree.
    await tester.tap(_crumb('Loop project'));
    await tester.pumpAndSettle();
    await _canvas(tester);
    expect(find.text('Search'), findsWidgets);
    expect(find.byKey(const Key('roadmap-breadcrumb')), findsNothing);
  });

  testWidgets('tapping a parent card opens its tree', (tester) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _canvas(tester);

    await tester.tap(find.text('Payments').first);
    await tester.pumpAndSettle();

    expect(
      router.routeInformationProvider.value.uri.path,
      '/projects/p1/roadmap/a',
    );
    await _canvas(tester);
    expect(find.text('Search'), findsNothing);
  });

  testWidgets('an ancestor crumb inside an opened tree opens that node', (
    tester,
  ) async {
    final api = TreeApi();
    final router = await pumpTreeRouter(tester, treeSession(api));
    router.go('/projects/p1/roadmap/a1');
    await tester.pumpAndSettle();

    // Same route pattern, new node id: the page must load the new subtree.
    await tester.tap(_crumb('Payments'));
    await tester.pumpAndSettle();

    expect(api.subtreeRoots, ['a1', 'a']);
    expect(_crumb('Loop project'), findsOneWidget);
    expect(_crumb('Ship'), findsOneWidget);
    expect(_crumb('Payments'), findsOneWidget);
    expect(_crumb('Refunds'), findsNothing);
    await _canvas(tester);
    expect(find.text('Refunds'), findsWidgets);
  });

  testWidgets('the node deep link renders after a cold start', (tester) async {
    tester.binding.platformDispatcher.defaultRouteNameTestValue =
        '/projects/p1/roadmap/a1';
    addTearDown(
      tester.binding.platformDispatcher.clearDefaultRouteNameTestValue,
    );
    final api = TreeApi();
    await pumpTreeRouter(tester, treeSession(api));

    expect(api.subtreeRoots, ['a1']);
    expect(_crumb('Ship'), findsOneWidget);
    expect(_crumb('Payments'), findsOneWidget);
  });

  testWidgets('a reader sees no New item; a plan writer does', (tester) async {
    final router = await pumpTreeRouter(
      tester,
      treeSession(TreeApi(canWritePlan: false, canWrite: false)),
    );
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    expect(find.text('New item'), findsNothing);

    final writer = await pumpTreeRouter(tester, treeSession(TreeApi()));
    writer.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    expect(find.text('New item'), findsOneWidget);
  });
}
