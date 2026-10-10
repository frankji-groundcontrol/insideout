import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// Pumps a signed-in shell with the dock on [initialLocation]. The session
/// remembers nothing by default; tests pass a seeded [LoopFocus] to model
/// warm (pages loaded ids) or cold (fresh dashboard) states.
Future<void> pumpDock(
  WidgetTester tester, {
  required LoopFocus focus,
  String initialLocation = '/projects/proj-1/roadmap',
}) async {
  final session = Session();
  session.ready = true;

  Widget page(String label) =>
      Scaffold(body: Text(label), bottomNavigationBar: const ProductDock());

  final router = GoRouter(
    initialLocation: initialLocation,
    routes: [
      GoRoute(path: '/dashboard', builder: (_, _) => page('Dashboard')),
      GoRoute(
        path: '/workspace/:id/ideas',
        builder: (_, s) => page('Ideas ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => page('Version ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/prd/:id/versions',
        builder: (_, s) => page('Commit ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/projects/:id/roadmap',
        builder: (_, s) => page('Roadmap ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/projects/:id',
        builder: (_, s) => page('Evidence ${s.pathParameters['id']}'),
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
        ChangeNotifierProvider.value(value: focus),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
}

/// Every id the dock can need, so all six items are enabled.
LoopFocus warmFocus({bool committed = true}) => LoopFocus()
  ..rememberWorkspace('ws-1')
  ..rememberProject('proj-1', workspaceId: 'ws-1')
  ..rememberPrd('prd-1', committed: committed);
