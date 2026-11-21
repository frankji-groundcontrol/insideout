import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEX-03: the signed-in landing lists the caller's own nodes (GET
// /workspaces/{id}/roadmap/mine per workspace) and each opens its tree.

class _MineApi extends TreeApi {
  /// The workspaces whose "mine" read was asked for.
  final mineReads = <String>[];

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
  Future<List<RoadmapNodeRead>> listMyRoadmap(String workspaceId) async {
    mineReads.add(workspaceId);
    return [await readRoadmapNode('a')];
  }
}

void main() {
  testWidgets('the dashboard lists my nodes and opens the right route', (
    tester,
  ) async {
    final api = _MineApi();
    final router = await pumpTreeRouter(
      tester,
      treeSession(api, userId: 'u-dana'),
    );
    router.go('/dashboard');
    await tester.pumpAndSettle();

    expect(api.mineReads, ['ws-1']);
    expect(find.text('My work'), findsOneWidget);
    expect(find.text('Payments'), findsOneWidget);
    expect(find.textContaining('Loop project'), findsOneWidget);
    expect(find.textContaining('Ship'), findsOneWidget);

    await tester.tap(find.text('Payments'));
    await tester.pumpAndSettle();
    expect(
      router.routeInformationProvider.value.uri.path,
      '/projects/p1/roadmap/a',
    );
  });
}
