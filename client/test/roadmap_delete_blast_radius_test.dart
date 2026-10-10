import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The server deletes a node's whole branch (roadmap_nodes.parent_id is
// ON DELETE CASCADE), so the confirm must say how many nodes go with it.

RoadmapNode _node(String id, String? parent, int position) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: id,
  description: '',
  status: 'pending',
  position: position,
);

/// Root > Child > Grandchild (two descendants), plus an untouched Leaf root.
final _tree = [
  _node('Root', null, 0),
  _node('Child', 'Root', 0),
  _node('Grandchild', 'Child', 0),
  _node('Leaf', null, 1),
];

class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final _parked = Completer<void>();

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => _tree;

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress.fromJson({
        'now': [],
        'needsDeadline': [],
        'next': [],
        'doneCount': 0,
      });

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _parked.future;

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _parked.future;
}

Future<void> _pumpPage(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _Api();
  await tester.binding.setSurfaceSize(const Size(1400, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
}

/// Opens [title]'s menu and picks Delete, leaving the confirm dialog up.
Future<void> _askToDelete(WidgetTester tester, String title) async {
  final card = find.widgetWithText(RoadmapNodeCard, title);
  await tester.tap(
    find.descendant(of: card, matching: find.byType(PopupMenuButton<String>)),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text('Delete').last);
  await tester.pumpAndSettle();
}

Future<void> _cancel(WidgetTester tester) async {
  await tester.tap(find.text('Cancel'));
  await tester.pumpAndSettle();
}

const _branchWording =
    'This deletes 3 nodes: this one and everything beneath it.';
const _leafWording = 'Delete this node and its whole branch?';

void main() {
  for (final view in ['Bands', 'Canvas']) {
    testWidgets('$view: deleting a branch root names every node it takes', (
      tester,
    ) async {
      await _pumpPage(tester);
      await tester.tap(find.text(view));
      await tester.pumpAndSettle();

      await _askToDelete(tester, 'Root');
      expect(find.text(_branchWording), findsOneWidget);
      await _cancel(tester);

      // A middle node counts only its own branch: itself and Grandchild.
      await _askToDelete(tester, 'Child');
      expect(
        find.text('This deletes 2 nodes: this one and everything beneath it.'),
        findsOneWidget,
      );
      await _cancel(tester);
    });

    testWidgets('$view: a node with nothing beneath keeps the leaf wording', (
      tester,
    ) async {
      await _pumpPage(tester);
      await tester.tap(find.text(view));
      await tester.pumpAndSettle();

      await _askToDelete(tester, 'Leaf');

      expect(find.text(_leafWording), findsOneWidget);
    });
  }
}
