import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

class _DraftApi extends ApiClient {
  _DraftApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  Map<String, dynamic>? saved;

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
  Future<Conversation?> conversationForPrd(String prdId) async => null;

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

  @override
  Future<Prd> updatePrd(
    String id, {
    String? title,
    Map<String, String>? sections,
  }) async {
    saved = {'id': id, 'title': title, 'sections': sections};
    return Prd(
      id: id,
      workspaceId: 'ws-1',
      title: title ?? 'Ship a thing',
      sections: sections ?? const {},
      status: 'draft',
      currentRevision: 1,
      updatedAt: '2026-10-05T00:00:00Z',
    );
  }
}

bool _enabled(WidgetTester tester, String label) {
  final button = tester.widget<BlockButton>(
    find.ancestor(of: find.text(label), matching: find.byType(BlockButton)),
  );
  return button.onPressed != null;
}

void main() {
  testWidgets('a draft PRD enables only its next action', (tester) async {
    final session = Session();
    session.ready = true;
    final api = _DraftApi();
    session.api = api;
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

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
    await tester.pump();

    expect(find.text('Next: Request review'), findsOneWidget);
    expect(_enabled(tester, 'Next: Request review'), isTrue);
    // A loaded draft can still be edited and saved. Save is not a status step.
    expect(_enabled(tester, 'Save'), isTrue);
    expect(_enabled(tester, 'Draft the roadmap'), isFalse);

    await tester.enterText(find.byType(TextField).at(1), 'A measurable goal');
    await tester.tap(find.text('Save'));
    await tester.pump();
    expect(api.saved?['id'], 'prd-1');
    expect((api.saved?['sections'] as Map)['background'], 'A measurable goal');

    await tester.tap(find.byTooltip('More'));
    await tester.pumpAndSettle();

    // The menu mirrors prdTransitions: a draft's one legal jump is reviewing.
    // The current status carries a square mark — dots are blocks now.
    final reviewing = tester.widget<PopupMenuItem<String>>(
      find.widgetWithText(PopupMenuItem<String>, 'In review'),
    );
    expect(reviewing.enabled, isTrue);
    for (final status in ['■ Draft', 'Approved', 'Rejected']) {
      final item = tester.widget<PopupMenuItem<String>>(
        find.widgetWithText(PopupMenuItem<String>, status),
      );
      expect(
        item.enabled,
        isFalse,
        reason: '$status is not the next step of a draft',
      );
    }
  });
}
