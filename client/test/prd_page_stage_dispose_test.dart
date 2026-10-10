import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The coach panel drops its own stream callbacks once it is gone, but a stage
// change also calls up into the PRD page, and the page's closure set state
// without asking whether the page was still there: a stage change that landed
// after the member left the PRD threw "setState() called after dispose()".
class _ConversationApi extends ApiClient {
  _ConversationApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

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

  // A conversation, or the page's stage callback has nothing to move.
  @override
  Future<Conversation?> conversationForPrd(String prdId) async =>
      Conversation(id: 'c1', prdId: prdId, stage: 'clarify', status: 'active');

  @override
  Future<List<ChatMessage>> listMessages(String conversationId) async =>
      const [];

  // The page looks up the viewer's role; keep it off the network.
  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Board Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });
}

Future<void> _pumpPage(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _ConversationApi();
  // The coach panel rides the wide desk (>= Cobalt.bpWide). setSurfaceSize
  // no longer reaches MediaQuery in this Flutter, so size the view itself.
  tester.view.physicalSize = const Size(1400, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
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
}

/// The very closure the page hands its coach panel for stage changes.
void Function(String stage) _stageCallback(WidgetTester tester) =>
    tester.widget<CoachPanel>(find.byType(CoachPanel)).onStageChanged;

void main() {
  testWidgets('a stage change that lands after the page is gone is dropped', (
    tester,
  ) async {
    await _pumpPage(tester);
    final onStage = _stageCallback(tester);

    await tester.pumpWidget(const SizedBox.shrink());

    expect(() => onStage('finalize'), returnsNormally);
    expect(tester.takeException(), isNull);
  });
}
