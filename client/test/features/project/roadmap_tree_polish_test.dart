import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// Wave 2 verifier follow-ups on the web. rv5-TREEX-03: Mine in an opened
// tree counts an owned ancestor. rv5-TREEX-08: an unowned node names the
// owner it inherits. rv5-TREEX-10: the bar reads its numbers, and My work shows no roll-up for
// a delegated leaf.

Future<void> _view(WidgetTester tester, String label) async {
  await tester.tap(find.text(label));
  await tester.pumpAndSettle();
}

class _LeafMineApi extends TreeApi {
  @override
  Future<List<Workspace>> listWorkspaces() async => [
    Workspace.fromJson({
      'id': 'ws-1',
      'title': 'Studio',
      'description': '',
      'code': 'CODE1',
      'status': 'active',
      'memberCount': 2,
      'myRole': 'member',
      'createdAt': '2026-11-01T00:00:00Z',
    }),
  ];

  @override
  Future<List<RoadmapNodeRead>> listMyRoadmap(String workspaceId) async => [
    RoadmapNodeRead(
      node: RoadmapNode(
        id: 'a1',
        projectId: 'p1',
        parentId: 'a',
        title: 'Refunds',
        description: '',
        status: 'pending',
        position: 0,
        ownerId: 'u-dana',
        ownerName: 'dana',
        rollup: const RoadmapRollup(leaves: 1),
      ),
      ancestors: const [RoadmapNodeRef(id: 'a', title: 'Payments')],
      projectId: 'p1',
      projectName: 'Loop project',
    ),
  ];
}

void main() {
  testWidgets('Mine in an opened tree below an owned ancestor shows it all', (
    tester,
  ) async {
    final router = await pumpTreeRouter(
      tester,
      treeSession(TreeApi(), userId: 'u-dana'),
    );
    router.go('/projects/p1/roadmap/a1');
    await tester.pumpAndSettle();
    await _view(tester, 'Mine');

    expect(find.text('Refunds'), findsWidgets);
    expect(find.text('Nothing in this tree is yours yet.'), findsNothing);
  });

  testWidgets('an unowned child names the owner it inherits', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await _view(tester, 'Canvas');

    expect(find.text('Owner: dana (via Payments)'), findsOneWidget);
    expect(find.text('Owner: dana'), findsOneWidget);

    // The leaf's detail says the same.
    await tester.tap(
      find.descendant(
        of: find.byKey(const Key('roadmap-graph-node-a1')),
        matching: find.byType(PopupMenuButton<String>),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Details'));
    await tester.pumpAndSettle();
    expect(find.text('dana (via Payments)'), findsOneWidget);
  });

  testWidgets('an opened tree inherits from an ancestor above it', (
    tester,
  ) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap/a1');
    await tester.pumpAndSettle();
    await _view(tester, 'Canvas');

    expect(find.text('Owner: dana (via Payments)'), findsOneWidget);
  });

  testWidgets('the roll-up bar reads its numbers', (tester) async {
    final parent = RoadmapNode(
      id: 'p',
      projectId: 'p1',
      title: 'Parent',
      description: '',
      status: 'in_progress',
      position: 0,
      canWrite: true,
      rollup: const RoadmapRollup(leaves: 3, done: 2),
    );
    await pumpTreeCard(tester, treeSession(TreeApi()), parent, descendants: 3);

    final bar = tester.widget<LinearProgressIndicator>(
      find.byType(LinearProgressIndicator),
    );
    expect(bar.semanticsLabel, '2 of 3 done');
  });

  testWidgets('My work shows no roll-up line for a delegated leaf', (
    tester,
  ) async {
    final router = await pumpTreeRouter(
      tester,
      treeSession(_LeafMineApi(), userId: 'u-dana'),
    );
    router.go('/dashboard');
    await tester.pumpAndSettle();

    expect(find.text('Refunds'), findsOneWidget);
    expect(find.textContaining('of 1 done'), findsNothing);
  });
}
