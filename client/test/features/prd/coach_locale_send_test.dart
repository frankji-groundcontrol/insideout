import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// FEAT-01 (review 3): the coach's prompt language is chosen at conversation
// creation, so the start request must carry the locale the app is already
// running in — Appearance.locale ('en-US' / 'zh-CN'), the same value the UI
// strings come from.

class _StartApi extends ApiClient {
  _StartApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// Every start request, as the real request builder shapes it.
  final calls = <ApiCall>[];

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: 'draft',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
    authorId: 'u-author',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Locale Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<Conversation> startPrdConversation(String prdId, String locale) async {
    calls.add(postPrdConversation(prdId, locale));
    return Conversation(
      id: 'c-new',
      prdId: prdId,
      stage: 'clarify',
      status: 'active',
    );
  }

  @override
  Future<List<ChatMessage>> listMessages(String conversationId) async =>
      const [];
}

Future<void> _mount(WidgetTester tester, _StartApi api, String locale) async {
  final session = Session()
    ..ready = true
    ..api = api
    ..user = UserProfile(
      id: 'u-viewer',
      email: 'a@b.c',
      username: 'tester',
      bio: '',
      keywords: const [],
    );
  // Below the wide breakpoint the coach column is never built.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = const Size(1400, 900);
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
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final (locale, start) in [
    ('en-US', 'Start coaching'),
    ('zh-CN', '开始教练对话'),
  ]) {
    testWidgets('starting a conversation sends the running locale ($locale)', (
      tester,
    ) async {
      final api = _StartApi();
      await _mount(tester, api, locale);

      await tester.tap(find.text(start));
      await tester.pumpAndSettle();

      expect(api.calls, hasLength(1));
      expect(api.calls.single.method, 'POST');
      expect(api.calls.single.path, '/prds/prd-1/conversation');
      expect(api.calls.single.body?['locale'], locale);
    });
  }
}
