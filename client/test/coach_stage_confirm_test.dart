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

// USER-01 (review 3): the coach's advance_stage tool only PROPOSES the next
// stage — it waits in conversation meta until POST /conversations/{id}/stage
// confirms it. That endpoint had no caller, and the page rendered the
// proposal as the current stage, so a refresh reverted it. These tests pin
// the member-facing half: a confirm/decline block while a proposal waits,
// and a masthead that keeps the confirmed stage until confirm lands.
class _StageApi extends ApiClient {
  _StageApi()
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

  @override
  Future<Conversation?> conversationForPrd(String prdId) async =>
      Conversation(id: 'c1', prdId: prdId, stage: 'clarify', status: 'active');

  /// The (conversationId, stage) pairs confirm posted, server-style: the
  /// stage column moves and the proposal is gone.
  final confirmedStages = <(String, String)>[];

  @override
  Future<Conversation> confirmConversationStage(
    String conversationId,
    String stage,
  ) async {
    confirmedStages.add((conversationId, stage));
    return Conversation(
      id: conversationId,
      prdId: 'prd-1',
      stage: stage,
      status: 'active',
    );
  }

  @override
  Future<List<ChatMessage>> listMessages(String conversationId) async =>
      const [];

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Stage Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });
}

Future<_StageApi> _pumpPage(WidgetTester tester) async {
  final api = _StageApi();
  final session = Session()
    ..ready = true
    ..api = api;
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
  return api;
}

/// The very closure the page hands its coach panel for stage_changed frames.
void Function(String stage) _stageCallback(WidgetTester tester) =>
    tester.widget<CoachPanel>(find.byType(CoachPanel)).onStageChanged;

void main() {
  testWidgets('confirm and decline appear only while a stage is proposed', (
    tester,
  ) async {
    await _pumpPage(tester);

    // A settled conversation proposes nothing.
    expect(find.byKey(const Key('coach-stage-confirm')), findsNothing);
    expect(find.byKey(const Key('coach-stage-decline')), findsNothing);

    _stageCallback(tester)('draft');
    await tester.pump();

    expect(find.byKey(const Key('coach-stage-confirm')), findsOneWidget);
    expect(find.byKey(const Key('coach-stage-decline')), findsOneWidget);
  });

  testWidgets('a proposed stage reads as proposed, not as current', (
    tester,
  ) async {
    await _pumpPage(tester);

    _stageCallback(tester)('draft');
    await tester.pump();

    // The masthead keeps the confirmed stage (the server's truth)...
    expect(find.text('Clarifying'), findsOneWidget);
    // ...and the proposal is labeled, so a refresh cannot revert it.
    expect(find.text('Proposed: Drafting'), findsOneWidget);
  });

  testWidgets('confirm posts the proposed stage and refreshes state', (
    tester,
  ) async {
    final api = await _pumpPage(tester);
    _stageCallback(tester)('draft');
    await tester.pump();

    await tester.tap(find.byKey(const Key('coach-stage-confirm')));
    await tester.pumpAndSettle();

    // The one and only POST named the conversation and the proposal.
    expect(api.confirmedStages, [('c1', 'draft')]);
    // The stage the server returns is now the current one.
    expect(find.text('Clarifying'), findsNothing);
    expect(find.text('Drafting'), findsOneWidget);
    expect(find.text('Proposed: Drafting'), findsNothing);
    expect(find.byKey(const Key('coach-stage-confirm')), findsNothing);
  });

  testWidgets('decline keeps the confirmed stage and clears the proposal', (
    tester,
  ) async {
    final api = await _pumpPage(tester);
    _stageCallback(tester)('draft');
    await tester.pump();

    await tester.tap(find.byKey(const Key('coach-stage-decline')));
    await tester.pump();

    // Declining is local: the endpoint expresses confirm only.
    expect(api.confirmedStages, isEmpty);
    expect(find.byKey(const Key('coach-stage-confirm')), findsNothing);
    expect(find.text('Clarifying'), findsOneWidget);
    expect(find.text('Proposed: Drafting'), findsNothing);
  });
}
