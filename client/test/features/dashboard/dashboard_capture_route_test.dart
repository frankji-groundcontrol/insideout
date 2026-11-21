import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/dashboard/dashboard_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// USER-09: with zero workspaces the dashboard's 'Capture an idea' CTA
// opened the create-workspace form and then stranded the user back on the
// dashboard holding the same CTA. The capture promise must complete in one
// gesture: create implied by capture, land on the new workspace's ideas
// wall where the capture bar is.
class _EmptyThenCreateApi extends ApiClient {
  _EmptyThenCreateApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<List<Workspace>> listWorkspaces() async => const [];

  @override
  Future<Workspace> createWorkspace(String title, String description) async =>
      Workspace.fromJson({
        'id': 'w-new',
        'title': title,
        'description': description,
        'code': 'NEW123',
        'status': 'active',
        'memberCount': 1,
        'myRole': 'admin',
        'createdAt': '2026-10-06T00:00:00Z',
      });
}

Future<void> _pump(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _EmptyThenCreateApi();
  final focus = LoopFocus();
  final router = GoRouter(
    initialLocation: '/dashboard',
    routes: [
      GoRoute(path: '/dashboard', builder: (_, _) => const DashboardPage()),
      GoRoute(
        path: '/workspace/:id/ideas',
        builder: (_, s) =>
            Scaffold(body: Text('Ideas ${s.pathParameters['id']}')),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider.value(value: focus),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'capture with zero workspaces lands on the new wall after creating',
    (tester) async {
      await _pump(tester);

      // The board is empty; the CTA implies the workspace it needs.
      await tester.tap(find.text('Capture an idea'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'First WS');
      await tester.tap(find.text('Create').last);
      await tester.pumpAndSettle();

      // The idea capture happens where the button promised: the new
      // workspace's ideas wall, not back on the dashboard.
      expect(find.text('Ideas w-new'), findsOneWidget);
      expect(find.text('Capture an idea'), findsNothing);
    },
  );
}
