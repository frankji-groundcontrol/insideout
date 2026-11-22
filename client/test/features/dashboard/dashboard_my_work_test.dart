import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';

import '../project/_expand_fixture_helpers.dart';
import '../project/_roadmap_tree_helpers.dart';

// People in charge (2026-11-22): the landing's "My work" reads every root the
// caller is in charge of in one GET /roadmap/mine, puts the pressed roots
// first, and shows each root's Now, what it waits on and what it blocks.

Map<String, dynamic> _node(String id, String title, String? parent) => {
  'id': id,
  'projectId': 'p1',
  'parentId': parent,
  'title': title,
  'status': 'in_progress',
};

final _mine = jsonEncode([
  {
    // An older server's root: no workspace, Now or links at all.
    'node': _node('r', 'Ship', null),
    'projectId': 'p1',
    'projectName': 'Loop project',
  },
  {
    'node': _node('a', 'Payments', 'r'),
    'projectId': 'p1',
    'workspaceId': 'ws-1',
    'projectName': 'Loop project',
    'ancestors': [
      {'id': 'r', 'title': 'Ship'},
    ],
    'now': [
      {
        'id': 'a1',
        'title': 'Refunds',
        'status': 'in_progress',
        'deadline': '2026-11-20T00:00:00Z',
        'pressure': 'overdue',
        'ownerName': 'dana',
      },
      {
        'id': 'a2',
        'title': 'Receipts',
        'status': 'in_progress',
        'pressure': 'normal',
      },
    ],
    'waitsOn': [
      {'id': 'b', 'title': 'Search', 'status': 'pending', 'ownerName': 'leo'},
      {'id': 'c', 'title': 'Ledger', 'status': 'pending'},
    ],
    'blocking': [
      {'id': 'd', 'title': 'Launch', 'status': 'pending', 'ownerName': 'sam'},
    ],
  },
]);

class _MineApi extends TreeApi {
  @override
  Future<List<Workspace>> listWorkspaces() async => [
    for (final id in ['ws-1', 'ws-2'])
      Workspace.fromJson({
        'id': id,
        'title': 'Studio $id',
        'description': '',
        'code': 'CODE1',
        'status': 'active',
        'memberCount': 2,
        'myRole': 'member',
        'createdAt': '2026-11-01T00:00:00Z',
      }),
  ];
}

Future<(GoRouter, ServerFixtures)> _landing(WidgetTester tester) async {
  final api = _MineApi();
  final server = ServerFixtures({'GET /api/v1/roadmap/mine': (200, _mine)});
  api.dioForTests.httpClientAdapter = server;
  final router = await pumpTreeRouter(
    tester,
    treeSession(api, userId: 'u-dana'),
  );
  router.go('/dashboard');
  await tester.pumpAndSettle();
  return (router, server);
}

void main() {
  testWidgets('one read across workspaces, the overdue root first', (
    tester,
  ) async {
    final (_, server) = await _landing(tester);

    expect(server.calls.where((c) => c.contains('mine')), [
      'GET /api/v1/roadmap/mine',
    ]);
    expect(find.text('My work'), findsOneWidget);
    final payments = tester.getTopLeft(find.text('Payments')).dy;
    expect(payments, lessThan(tester.getTopLeft(find.text('Ship')).dy));
  });

  testWidgets('a root shows its Now with pressure, waits and blocking', (
    tester,
  ) async {
    await _landing(tester);

    expect(find.text('Refunds'), findsOneWidget);
    expect(find.text('overdue'), findsOneWidget);
    expect(find.text('Receipts'), findsOneWidget);
    expect(find.text('normal'), findsNothing);
    expect(find.text('Waits on: Search (leo), Ledger'), findsOneWidget);
    expect(find.text('Blocking: Launch (sam)'), findsOneWidget);
    // A root with nothing waiting or blocked says neither.
    expect(find.textContaining('Waits on'), findsOneWidget);
    expect(find.textContaining('Blocking'), findsOneWidget);
  });

  testWidgets('a root opens its Canvas; the project root the project roadmap', (
    tester,
  ) async {
    final (router, _) = await _landing(tester);

    await tester.tap(find.text('Payments'));
    await tester.pumpAndSettle();
    final opened = router.routeInformationProvider.value.uri;
    expect(opened.path, '/projects/p1/roadmap/a');
    expect(opened.queryParameters['view'], 'canvas');

    router.go('/dashboard');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Ship'));
    await tester.pumpAndSettle();
    final root = router.routeInformationProvider.value.uri;
    expect(root.path, '/projects/p1/roadmap');
    expect(root.queryParameters['view'], 'canvas');
  });
}
