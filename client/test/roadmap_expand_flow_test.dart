import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

import '_expand_fixture_helpers.dart';

// rv4-FULL-01, the UI half: the node menu's "Break down with AI" drafts
// subtasks, SHOWS the draft, and adds them only when the person confirms
// (the server's draft route writes nothing; its confirm route writes). The
// client is the real ApiClient over the responses a running server sent.

final _node = RoadmapNode(
  id: 'n1',
  projectId: 'p1',
  title: 'Ship the first version',
  description: '',
  status: 'pending',
  position: 0,
);

Future<void> _pumpCard(
  WidgetTester tester,
  ServerFixtures server, {
  required void Function() onReload,
  String locale = 'en-US',
}) async {
  final session = Session()
    ..ready = true
    ..api = clientOver(server);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
      ],
      child: MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: RoadmapNodeCard(
              projectId: 'p1',
              node: _node,
              onReload: () async => onReload(),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

/// Opens the node's menu and taps the AI breakdown entry.
Future<void> _breakDown(WidgetTester tester, String label) async {
  await tester.tap(find.byType(PopupMenuButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text(label));
  await tester.pumpAndSettle();
}

const _draftCall = 'POST /api/v1/roadmap/n1/expand';
const _confirmCall = 'POST /api/v1/roadmap/n1/expand/confirm';

void main() {
  testWidgets('the draft is shown, and confirming adds exactly the draft', (
    tester,
  ) async {
    final sent = capturedDraftChildren();
    final server = ServerFixtures.captured('n1');
    var reloads = 0;
    await _pumpCard(tester, server, onReload: () => reloads++);

    await _breakDown(tester, 'Break down with AI');

    // The person sees every drafted subtask before anything is written.
    for (final child in sent) {
      expect(find.textContaining(child['title'] as String), findsOneWidget);
    }
    expect(server.calls, [_draftCall]);
    expect(reloads, 0);

    await tester.tap(find.text('Add ${sent.length} sub-tasks'));
    await tester.pumpAndSettle();

    expect(server.calls, [_draftCall, _confirmCall]);
    expect(server.requests.last.data, {'children': sent});
    expect(reloads, 1);
  });

  testWidgets('cancelling the review adds nothing', (tester) async {
    final server = ServerFixtures.captured('n1');
    var reloads = 0;
    await _pumpCard(tester, server, onReload: () => reloads++);

    await _breakDown(tester, 'Break down with AI');
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(server.calls, [_draftCall]);
    expect(reloads, 0);
  });

  testWidgets('an empty draft says so instead of offering a confirm', (
    tester,
  ) async {
    final server = ServerFixtures({_draftCall: (200, '{"children":[]}')});
    await _pumpCard(tester, server, onReload: () {});

    await _breakDown(tester, 'Break down with AI');

    expect(
      find.text('The AI had no sub-tasks to suggest for this node.'),
      findsOneWidget,
    );
    expect(find.text('Cancel'), findsNothing);
    expect(server.calls, [_draftCall]);
  });

  testWidgets('the review speaks zh-CN', (tester) async {
    final sent = capturedDraftChildren();
    final server = ServerFixtures.captured('n1');
    await _pumpCard(tester, server, onReload: () {}, locale: 'zh-CN');

    await _breakDown(tester, '用 AI 拆解');

    expect(find.text('添加 ${sent.length} 个子任务'), findsOneWidget);
    expect(find.text('取消'), findsOneWidget);
  });
}
