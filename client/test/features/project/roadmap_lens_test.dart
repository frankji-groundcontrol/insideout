import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/theme/cobalt_chip.dart';

import '_roadmap_canvas_helpers.dart';
import '_roadmap_tree_helpers.dart';

// View as the person in charge (2026-11-22): an opened node shows whose view
// it is, with that person's first glance at the node (Now, waits on,
// blocking), and the fourth toggle shows their own nodes.

Map<String, dynamic> _glance({
  String userId = 'u-dana',
  Map<String, dynamic>? via,
  bool projectOwner = false,
  bool quiet = false,
}) => {
  'node': {'id': 'a', 'projectId': 'p1', 'title': 'Payments'},
  'inCharge': {
    'userId': userId,
    'name': 'Priya Nair',
    'via': via,
    'projectOwner': projectOwner,
  },
  'now': [
    if (!quiet) ...[
      {
        'id': 'a1',
        'title': 'Refunds',
        'status': 'in_progress',
        'pressure': 'near',
      },
      {
        'id': 'x2',
        'title': 'Receipts',
        'status': 'in_progress',
        'pressure': 'normal',
      },
      {
        'id': 'x3',
        'title': 'Payouts',
        'status': 'in_progress',
        'pressure': 'overdue',
      },
    ],
  ],
  'waitsOn': [
    if (!quiet)
      {'id': 'b', 'title': 'Search', 'status': 'pending', 'ownerName': 'leo'},
  ],
  'blocking': [
    if (!quiet) {'id': 'd', 'title': 'Launch', 'status': 'pending'},
  ],
};

Future<(GoRouter, TreeApi)> _open(
  WidgetTester tester,
  String path, {
  Map<String, dynamic>? glance,
  String userId = 'u-me',
}) async {
  final api = TreeApi()..glance = glance;
  final router = await pumpTreeRouter(tester, treeSession(api, userId: userId));
  router.go(path);
  await tester.pumpAndSettle();
  return (router, api);
}

ShadeChip _chip(WidgetTester tester, String label) => tester.widget<ShadeChip>(
  find.byWidgetPredicate((w) => w is ShadeChip && w.label == label),
);

void main() {
  test('a glance an older server leaves short reads as empty', () {
    final g = RoadmapGlance.fromJson(const {});
    expect(g.inCharge, isNull);
    expect(g.now, isEmpty);
    expect(g.waitsOn, isEmpty);
    expect(g.blocking, isEmpty);
    final full = RoadmapGlance.fromJson(
      _glance(via: {'id': 'r', 'title': 'Ship'}),
    );
    expect(full.inCharge!.via?.title, 'Ship');
    expect(full.inCharge!.firstName, 'Priya');
  });

  testWidgets('an opened node says whose view it is, with their glance', (
    tester,
  ) async {
    final (_, api) = await _open(
      tester,
      '/projects/p1/roadmap/a',
      glance: _glance(),
    );

    expect(api.glanceCalls, ['a']);
    expect(find.text("PRIYA NAIR'S VIEW · in charge"), findsOneWidget);
    expect(find.text('Refunds'), findsOneWidget);
    expect(_chip(tester, 'near').tone, ShadeTone.neutral);
    expect(_chip(tester, 'overdue').tone, ShadeTone.danger);
    expect(find.text('normal'), findsNothing);
    expect(find.text('Waits on: Search (leo)'), findsOneWidget);
    expect(find.text('Blocking: Launch'), findsOneWidget);
    expect(find.text("Priya's"), findsOneWidget);
  });

  testWidgets('a node you are in charge of reads as your view', (tester) async {
    await _open(
      tester,
      '/projects/p1/roadmap/a',
      glance: _glance(userId: 'u-me'),
    );

    expect(find.text('YOUR VIEW · in charge'), findsOneWidget);
    expect(find.text("Priya's"), findsNothing);
    expect(find.text('Mine'), findsOneWidget);
  });

  testWidgets(
    'inherited charge names the ancestor; the project owner says so',
    (tester) async {
      await _open(
        tester,
        '/projects/p1/roadmap/a1',
        glance: _glance(via: {'id': 'a', 'title': 'Payments'}),
      );
      expect(
        find.text("PRIYA NAIR'S VIEW · in charge · via Payments"),
        findsOneWidget,
      );

      await _open(
        tester,
        '/projects/p1/roadmap/a1',
        glance: _glance(projectOwner: true),
      );
      expect(
        find.text("PRIYA NAIR'S VIEW · in charge · project owner"),
        findsOneWidget,
      );
    },
  );

  testWidgets('an empty glance reads one quiet line', (tester) async {
    await _open(tester, '/projects/p1/roadmap/a', glance: _glance(quiet: true));

    expect(find.text('Nothing due, waiting or blocked here.'), findsOneWidget);
    expect(find.textContaining('Waits on:'), findsNothing);
    expect(find.textContaining('Blocking:'), findsNothing);
  });

  testWidgets('a Now item opens its detail', (tester) async {
    await _open(tester, '/projects/p1/roadmap/a', glance: _glance());

    await tester.tap(find.text('Refunds'));
    await tester.pumpAndSettle();
    expect(find.text('Assignment history'), findsOneWidget);
  });

  testWidgets('the project roadmap has no banner and keeps Mine', (
    tester,
  ) async {
    final (_, api) = await _open(
      tester,
      '/projects/p1/roadmap',
      glance: _glance(),
    );

    expect(api.glanceCalls, isEmpty);
    expect(find.textContaining('VIEW ·'), findsNothing);
    expect(find.text('Mine'), findsOneWidget);
  });

  testWidgets('a failed glance shows no banner and keeps Mine', (tester) async {
    await _open(tester, '/projects/p1/roadmap/a');

    expect(find.textContaining('VIEW ·'), findsNothing);
    expect(find.text('Mine'), findsOneWidget);
  });

  testWidgets("the fourth toggle shows the person in charge's nodes", (
    tester,
  ) async {
    await _open(tester, '/projects/p1/roadmap/r', glance: _glance());

    await tester.tap(find.text("Priya's"));
    await tester.pumpAndSettle();
    // u-dana owns Payments alone: it is their one topmost node.
    expect(stepBox('a'), findsOneWidget);
    expect(stepBox('b'), findsNothing);
  });

  testWidgets('a level narrower than the view opens centred across it', (
    tester,
  ) async {
    // Twelve steps with no links stand in one tall column.
    await pumpCanvas(tester, [
      canvasNode('R', null),
      for (var i = 0; i < 12; i++) canvasNode('S$i', 'R', position: i),
    ], size: const Size(1400, 600));
    final view = tester.getRect(find.byType(InteractiveViewer));
    final card = stepCard(tester, 'S0');
    expect(card.left - view.left, closeTo(view.right - card.right, 2));
  });
}
