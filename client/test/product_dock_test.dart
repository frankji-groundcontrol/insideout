import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// A project whose working version has no human commit. The dock should name
/// Commit as the unfinished step and stay on this project even when the
/// project id was never remembered.
Future<void> _pump(WidgetTester tester, {required LoopFocus focus}) async {
  final session = Session();
  session.ready = true;

  Widget page(String label) => Scaffold(body: Text(label), bottomNavigationBar: const ProductDock());

  final router = GoRouter(
    initialLocation: '/projects/proj-1/roadmap',
    routes: [
      GoRoute(path: '/dashboard', builder: (_, _) => page('Dashboard')),
      GoRoute(path: '/workspace/:id/ideas', builder: (_, s) => page('Ideas ${s.pathParameters['id']}')),
      GoRoute(path: '/prd/:id', builder: (_, s) => page('Version ${s.pathParameters['id']}')),
      GoRoute(path: '/prd/:id/versions', builder: (_, s) => page('Commit ${s.pathParameters['id']}')),
      GoRoute(path: '/projects/:id/roadmap', builder: (_, s) => page('Roadmap ${s.pathParameters['id']}')),
      GoRoute(path: '/projects/:id', builder: (_, s) => page('Evidence ${s.pathParameters['id']}')),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ChangeNotifierProvider.value(value: focus),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
}

void main() {
  testWidgets('an uncommitted PRD marks Commit unfinished and a tap stays on the project', (tester) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    // The page has loaded the working version and recorded that nobody has
    // committed it. The project id itself was never remembered, which is the
    // case that used to fall through to the dashboard.
    final focus = LoopFocus()..rememberPrd('prd-1', committed: false);
    await _pump(tester, focus: focus);

    expect(find.byKey(const ValueKey('dock-unfinished-commit')), findsOneWidget);
    expect(find.text('Unfinished: Commit'), findsOneWidget);
    expect(find.text('Dashboard'), findsNothing);

    await tester.tap(find.text('Unfinished: Commit'));
    await tester.pumpAndSettle();

    expect(find.text('Dashboard'), findsNothing);
    expect(find.text('Commit prd-1'), findsOneWidget);
  });

  testWidgets('a commit tap with no remembered id stays on the project', (tester) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    // Neither the project nor the PRD was remembered. The route is already
    // the project, so Commit stays there instead of dropping to the dashboard.
    final focus = LoopFocus()..prdCommitted = false;
    await _pump(tester, focus: focus);

    expect(find.text('Roadmap proj-1'), findsOneWidget);
    await tester.tap(find.text('Unfinished: Commit'));
    await tester.pumpAndSettle();

    expect(find.text('Dashboard'), findsNothing);
    expect(find.text('Commit prd-1'), findsNothing);
    expect(find.text('Evidence proj-1'), findsOneWidget);
  });
}
