import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-01: during a session the transcript showed only the latest streamed
// reply: the member's own message never joined the list and the previous
// reply was wiped on the next send. Two turns must leave four bubbles on
// screen with no reload.
class _ChatApi extends ApiClient {
  _ChatApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// What the server has stored, as listMessages answers it.
  final stored = <ChatMessage>[];
  int turns = 0;
  // Network down: nothing reaches the server and nothing can be re-read.
  bool offline = false;

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: 'draft',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async =>
      Conversation(id: 'c1', prdId: prdId, stage: 'clarify', status: 'active');

  @override
  Future<Conversation> getConversation(String id) async {
    if (offline) throw Exception('offline');
    return Conversation(
      id: id,
      prdId: 'prd-1',
      stage: 'clarify',
      status: 'active',
    );
  }

  @override
  Future<List<ChatMessage>> listMessages(String conversationId) async {
    if (offline) throw Exception('offline');
    return List.of(stored);
  }

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Transcript WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    if (offline) throw Exception('offline');
    turns++;
    final reply = 'Reply number $turns';
    stored.add(ChatMessage(id: 'u$turns', role: 'user', content: content));
    handlers.onMessageStart?.call(false);
    handlers.onDelta?.call(reply);
    stored.add(ChatMessage(id: 'a$turns', role: 'assistant', content: reply));
  }
}

Future<_ChatApi> _pump(WidgetTester tester) async {
  final api = _ChatApi();
  final session = Session()
    ..ready = true
    ..api = api;
  tester.view.physicalSize = const Size(1400, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  final router = GoRouter(
    initialLocation: '/prd/prd-1',
    routes: [
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => PrdPage(id: s.pathParameters['id']!),
      ),
    ],
  );
  addTearDown(router.dispose);
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
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

Future<void> _say(WidgetTester tester, String text) async {
  await tester.enterText(find.byType(TextField).last, text);
  await tester.tap(find.byTooltip('Send'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('two turns leave both questions and both replies on screen', (
    tester,
  ) async {
    await _pump(tester);

    await _say(tester, 'First question');
    await _say(tester, 'Second question');

    for (final line in [
      'First question',
      'Reply number 1',
      'Second question',
      'Reply number 2',
    ]) {
      expect(find.text('“$line”'), findsOneWidget, reason: line);
    }
  });

  testWidgets('a send that never arrives leaves no sent bubble behind', (
    tester,
  ) async {
    final api = await _pump(tester);
    api.offline = true;

    await _say(tester, 'Lost question');

    expect(find.text('“Lost question”'), findsNothing);
    final composer = tester.widget<TextField>(find.byType(TextField).last);
    expect(composer.controller?.text, 'Lost question');
  });
}
