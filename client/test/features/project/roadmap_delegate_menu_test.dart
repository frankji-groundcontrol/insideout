import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '_roadmap_tree_helpers.dart';

// rv5-TREEX-08, TREEX-07, FE-05, TREEX-11: the node menu delegates and hands
// back on the record, offers write items only where canWrite says they
// succeed, and opens a read-only detail.

const _writeItems = [
  'Edit',
  'Add sub-task',
  'Move up',
  'Move down',
  'Break down with AI',
  'Delete',
  'Delegate to…',
];

void main() {
  testWidgets('delegating posts the chosen member and the note', (
    tester,
  ) async {
    final api = TreeApi();
    await pumpTreeCard(tester, treeSession(api), api.all[2]);
    await openNodeMenu(tester);
    await tester.tap(find.text('Delegate to…'));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('delegate-picker')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('sam').last);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('delegate-note')),
      'refunds are yours',
    );
    await tester.tap(find.text('Delegate'));
    await tester.pumpAndSettle();

    expect(api.delegated?.nodeId, 'a1');
    expect(api.delegated?.toUserId, 'u-sam');
    expect(api.delegated?.note, 'refunds are yours');
  });

  testWidgets('hand back appears only for the node owner', (tester) async {
    final api = TreeApi();
    final payments = api.all[1]; // owned by u-dana

    await pumpTreeCard(tester, treeSession(api), payments);
    await openNodeMenu(tester);
    expect(find.text('Hand back'), findsNothing);
    await tester.tapAt(Offset.zero);
    await tester.pumpAndSettle();

    await pumpTreeCard(tester, treeSession(api, userId: 'u-dana'), payments);
    await openNodeMenu(tester);
    await tester.tap(find.text('Hand back'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Hand back').last);
    await tester.pumpAndSettle();

    expect(api.handedBack?.nodeId, 'a');
  });

  testWidgets('a node with canWrite false shows no write items', (
    tester,
  ) async {
    final api = TreeApi(canWritePlan: false, canWrite: false);
    await pumpTreeCard(tester, treeSession(api), api.all[1], planWriter: false);
    await openNodeMenu(tester);

    for (final item in _writeItems) {
      expect(find.text(item), findsNothing, reason: item);
    }
    expect(find.text('Open as tree'), findsOneWidget);
    expect(find.text('Details'), findsOneWidget);
  });

  testWidgets('a delegate edits their root but cannot move, delete or re-own '
      'it', (tester) async {
    final api = TreeApi(canWritePlan: false);
    await pumpTreeCard(
      tester,
      treeSession(api, userId: 'u-dana'),
      api.all[1],
      planWriter: false,
    );
    await openNodeMenu(tester);

    expect(find.text('Edit'), findsOneWidget);
    expect(find.text('Add sub-task'), findsOneWidget);
    expect(find.text('Delete'), findsNothing);
    expect(find.text('Move up'), findsNothing);
    expect(find.text('Delegate to…'), findsNothing);
  });

  testWidgets('details show description, evidence and assignment history', (
    tester,
  ) async {
    final api = TreeApi(canWrite: false);
    await pumpTreeCard(tester, treeSession(api), api.all[1]);
    await openNodeMenu(tester);
    await tester.tap(find.text('Details'));
    await tester.pumpAndSettle();

    expect(find.text('Take money reliably'), findsOneWidget);
    expect(find.text('retry the charge once'), findsOneWidget);
    expect(find.textContaining('mira delegated to dana'), findsOneWidget);
    expect(find.textContaining('yours for the sprint'), findsOneWidget);
  });
}
