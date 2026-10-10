import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// A coach conversation is owner-only and ConvertIdea makes just the
// converter's, so every other teammate opens the PRD with none. The panel then
// showed one sentence and no composer: a dead end. It now offers to start one,
// to the people the server lets (the PRD's author or a workspace admin).
const _sentence = 'No coach conversation is linked to this PRD yet.';
const _composerHint = 'Type your reply...';

class _StartApi extends ApiClient {
  _StartApi({
    this.role = 'admin',
    this.failure,
    this.hold,
    this.existing = const [],
  }) : super(
         baseUrl: 'http://127.0.0.1:9/api/v1',
         readAccess: () async => null,
         readRefresh: () async => null,
         writeTokens: (a, r) async {},
         clearTokens: () async {},
       );

  /// The viewer's role in the PRD's workspace.
  final String role;

  /// Thrown by the start call, when set.
  final Object? failure;

  /// The start call waits for this before it answers, when set.
  final Completer<void>? hold;

  /// What the started conversation already holds (a resumed one).
  final List<ChatMessage> existing;

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
    'title': 'Board Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': role,
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<Conversation> startPrdConversation(String prdId) async {
    calls.add(postPrdConversation(prdId));
    await hold?.future;
    final failure = this.failure;
    if (failure != null) throw failure;
    return Conversation(
      id: 'c-new',
      prdId: prdId,
      stage: 'clarify',
      status: 'active',
    );
  }

  @override
  Future<List<ChatMessage>> listMessages(String conversationId) async =>
      existing;
}

/// Opens the PRD page as [viewer] (the signed-in user), with the coach column
/// beside the desk.
Future<void> _mount(
  WidgetTester tester,
  _StartApi api, {
  String viewer = 'u-viewer',
  String locale = 'en-US',
}) async {
  final session = Session()
    ..ready = true
    ..api = api
    ..user = UserProfile(
      id: viewer,
      email: 'a@b.c',
      username: 'tester',
      bio: '',
      keywords: const [],
    );
  // setSurfaceSize leaves MediaQuery at 800x600, below the wide breakpoint,
  // where the coach sits under the desk and is never built: resize the view.
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
  testWidgets(
    'a PRD with no conversation offers to start one, with no composer',
    (tester) async {
      await _mount(tester, _StartApi());

      expect(find.text(_sentence), findsOneWidget);
      expect(find.text('Start coaching'), findsOneWidget);
      expect(find.text(_composerHint), findsNothing);
    },
  );

  testWidgets('Start coaching posts to the PRD and opens the composer', (
    tester,
  ) async {
    final api = _StartApi();
    await _mount(tester, api);

    await tester.tap(find.text('Start coaching'));
    await tester.pumpAndSettle();

    expect(api.calls, hasLength(1));
    expect(api.calls.single.method, 'POST');
    expect(api.calls.single.path, '/prds/prd-1/conversation');
    expect(find.text(_composerHint), findsOneWidget);
    expect(find.text('Start coaching'), findsNothing);
    expect(find.text(_sentence), findsNothing);
  });

  testWidgets('the button reads in Chinese too', (tester) async {
    await _mount(tester, _StartApi(), locale: 'zh-CN');

    expect(find.text('开始教练对话'), findsOneWidget);
  });

  testWidgets('a second tap while the start is in flight sends one request', (
    tester,
  ) async {
    final hold = Completer<void>();
    final api = _StartApi(hold: hold);
    await _mount(tester, api);

    await tester.tap(find.text('Start coaching'));
    await tester.pump();
    await tester.tap(find.text('Start coaching'));
    await tester.pump();
    expect(api.calls, hasLength(1));

    hold.complete();
    await tester.pumpAndSettle();
    expect(api.calls, hasLength(1));
    expect(find.text(_composerHint), findsOneWidget);
  });

  testWidgets('a failed start says so and keeps the button for another try', (
    tester,
  ) async {
    final api = _StartApi(failure: ApiException('nope', status: 403));
    await _mount(tester, api);

    await tester.tap(find.text('Start coaching'));
    await tester.pumpAndSettle();

    expect(find.byType(SnackBar), findsOneWidget);
    expect(find.text('Start coaching'), findsOneWidget);
    expect(find.text(_composerHint), findsNothing);
  });

  testWidgets('a resumed conversation shows what it already holds', (
    tester,
  ) async {
    final api = _StartApi(
      existing: [
        ChatMessage(id: 'm1', role: 'user', content: 'We are two people'),
      ],
    );
    await _mount(tester, api);

    await tester.tap(find.text('Start coaching'));
    await tester.pumpAndSettle();

    expect(find.text('We are two people'), findsOneWidget);
  });

  testWidgets(
    'a plain member who did not write the PRD keeps only the sentence',
    (tester) async {
      await _mount(tester, _StartApi(role: 'member'));

      expect(find.text(_sentence), findsOneWidget);
      expect(find.text('Start coaching'), findsNothing);
    },
  );

  testWidgets('the author may start one even as a plain member', (
    tester,
  ) async {
    await _mount(tester, _StartApi(role: 'member'), viewer: 'u-author');

    expect(find.text('Start coaching'), findsOneWidget);
  });
}
