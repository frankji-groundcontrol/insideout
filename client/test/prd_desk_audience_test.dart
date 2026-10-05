import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
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
}

void main() {
  testWidgets('the audience action is visible without opening a menu', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _DraftApi();
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final router = GoRouter(
      initialLocation: '/prd/prd-1',
      routes: [
        GoRoute(path: '/prd/:id', builder: (_, s) => PrdPage(id: s.pathParameters['id']!)),
        GoRoute(path: '/prd/:id/view', builder: (_, __) => const Text('audience page')),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: MaterialApp.router(routerConfig: router),
      ),
    );
    await tester.pump();

    expect(find.byType(PopupMenuItem<String>), findsNothing);
    final button = tester.widget<ButtonStyleButton>(
      find.ancestor(of: find.text('Audience view'), matching: find.byWidgetPredicate((w) => w is ButtonStyleButton)),
    );
    expect(button.onPressed, isNotNull);

    await tester.tap(find.text('Audience view'));
    await tester.pumpAndSettle();
    expect(find.text('audience page'), findsOneWidget);
  });
}
