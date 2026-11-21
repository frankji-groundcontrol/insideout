import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_tree_helpers.dart';

// Roadmap critique 2026-11-21: the pinned header took 190px on desktop, an
// opened branch was still titled "ROADMAP", and presence repeated one user
// per session.

class _LongApi extends TreeApi {
  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(
        now: [
          for (var i = 0; i < 24; i++)
            ProgressItem(id: 'p$i', title: 'Item $i', status: 'in_progress'),
        ],
        needsDeadline: const [],
        next: const [],
        doneCount: 0,
      );

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) {
    PresenceEntry e(String s, String u, String n) =>
        PresenceEntry(sessionId: s, userId: u, name: n);
    onChange([
      e('s1', 'u-dana', 'dana'),
      e('s2', 'u-dana', 'dana'),
      e('s3', 'u-sam', 'sam'),
    ]);
    return super.presenceStream(projectId, clientId, onChange);
  }
}

final _title = find.byKey(const Key('roadmap-title'));

void main() {
  testWidgets('the header scrolls away with the Progress content', (
    tester,
  ) async {
    final router = await pumpTreeRouter(tester, treeSession(_LongApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();

    final before = tester.getRect(_title).top;
    await tester.drag(find.text('Item 3'), const Offset(0, -150));
    await tester.pumpAndSettle();
    expect(tester.getRect(_title).top, lessThan(before - 100));
  });

  testWidgets('the header scrolls away with the Bands content', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(_LongApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bands'));
    await tester.pumpAndSettle();

    final before = tester.getRect(_title).top;
    await tester.drag(find.text('LATER'), const Offset(0, -150));
    await tester.pumpAndSettle();
    expect(tester.getRect(_title).top, lessThan(before - 100));
  });

  testWidgets('an opened branch is titled with its node', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap/a');
    await tester.pumpAndSettle();

    expect(tester.widget<Text>(_title).data, 'Payments');
    expect(find.text('ROADMAP'), findsNothing);

    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    expect(tester.widget<Text>(_title).data, 'ROADMAP');
  });

  testWidgets('presence shows one chip per user', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(_LongApi()));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();

    expect(find.text('dana'), findsOneWidget);
    expect(find.text('sam'), findsOneWidget);
  });
}
