import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// 2026-11-22: "Depends on…" in the node menu links a node to one it waits on,
// anywhere in the project, and drops a link. The tree is Ship > {Payments >
// Refunds, Search}; Search's own ancestor (Ship) is no pick, since the tree
// already says that, and a refused link reads as its reason.

RoadmapNode _search({List<String> dependsOn = const []}) => RoadmapNode(
  id: 'b',
  projectId: 'p1',
  parentId: 'r',
  title: 'Search',
  description: '',
  status: 'pending',
  position: 0,
  canWrite: true,
  dependsOn: dependsOn,
);

Future<void> _openPicker(WidgetTester tester, TreeApi api, RoadmapNode node) =>
    pumpTreeCard(tester, treeSession(api), node).then((_) async {
      await openNodeMenu(tester);
      await tester.tap(find.text('Depends on…'));
      await tester.pumpAndSettle();
    });

Finder _option(String id) => find.byKey(Key('dependency-option-$id'));

void main() {
  testWidgets('links a node to the one picked by title', (tester) async {
    final api = TreeApi();
    await _openPicker(tester, api, _search());

    expect(_option('a'), findsOneWidget);
    expect(_option('a1'), findsOneWidget);
    expect(_option('r'), findsNothing, reason: 'Ship is above Search');
    expect(_option('b'), findsNothing, reason: 'Search cannot wait on itself');

    await tester.enterText(find.byType(TextField), 'ref');
    await tester.pumpAndSettle();
    expect(_option('a'), findsNothing);

    await tester.tap(_option('a1'));
    await tester.pumpAndSettle();
    expect(api.dependencyCalls, [(nodeId: 'b', on: 'a1', remove: false)]);
  });

  testWidgets('a refused loop reads as its reason', (tester) async {
    final api = TreeApi()
      ..dependencyError = ApiException(
        'cycle',
        status: 409,
        code: 'dependency_cycle',
      );
    await _openPicker(tester, api, _search());
    await tester.tap(_option('a'));
    await tester.pumpAndSettle();

    expect(find.textContaining('would close a loop'), findsOneWidget);
  });

  testWidgets('drops a link the node already has', (tester) async {
    final api = TreeApi();
    await _openPicker(tester, api, _search(dependsOn: ['a']));

    // Payments is a current link, so it is offered for removal, not again.
    expect(_option('a'), findsNothing);
    await tester.tap(find.bySemanticsLabel('Stop waiting on Payments'));
    await tester.pumpAndSettle();
    expect(api.dependencyCalls, [(nodeId: 'b', on: 'a', remove: true)]);
  });

  testWidgets('a reader sees no Depends on item', (tester) async {
    await pumpTreeCard(
      tester,
      treeSession(TreeApi(canWrite: false)),
      RoadmapNode(
        id: 'b',
        projectId: 'p1',
        parentId: 'r',
        title: 'Search',
        description: '',
        status: 'pending',
        position: 0,
      ),
    );
    await openNodeMenu(tester);
    expect(find.text('Depends on…'), findsNothing);
  });
}
