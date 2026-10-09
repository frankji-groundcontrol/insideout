import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/app.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:provider/provider.dart';

/// AppScaffold-shell coverage split out of `product_dock_test.dart`: the
/// extendBody dock clearance and the top-right utility cluster are chrome
/// concerns, not dock concerns.
///
/// Every id the dock can need, so all six dock items are enabled.
LoopFocus warmFocus({bool committed = true}) => LoopFocus()
  ..rememberWorkspace('ws-1')
  ..rememberProject('proj-1', workspaceId: 'ws-1')
  ..rememberPrd('prd-1', committed: committed);

void main() {
  testWidgets(
    'extendBody scaffold at 1400x900 keeps the dock off the body landmark',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final session = Session()
        ..ready = true
        ..user = UserProfile(
          id: 'u1',
          email: 'a@b.c',
          username: 'tester',
          bio: '',
          keywords: const [],
        );
      final router = GoRouter(
        initialLocation: '/projects/proj-1',
        routes: [
          GoRoute(
            path: '/projects/:id',
            builder: (_, _) => const AppScaffold(
              body: Center(
                child: Text('landmark', key: ValueKey('body-landmark')),
              ),
            ),
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
            ChangeNotifierProvider.value(value: warmFocus()),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      expect(tester.widget<Scaffold>(find.byType(Scaffold)).extendBody, isTrue);

      final dockRect = tester.getRect(find.byType(ProductDock));
      final markRect = tester.getRect(
        find.byKey(const ValueKey('body-landmark')),
      );
      expect(dockRect.overlaps(markRect), isFalse);
      expect(dockRect.top, greaterThan(markRect.bottom));
    },
  );

  testWidgets(
    'the utility cluster is a bordered cast block of BlockIconButtons',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final session = Session()
        ..ready = true
        ..user = UserProfile(
          id: 'u1',
          email: 'a@b.c',
          username: 'tester',
          bio: '',
          keywords: const [],
        );
      final router = GoRouter(
        initialLocation: '/projects/proj-1',
        routes: [
          GoRoute(
            path: '/projects/:id',
            builder: (_, _) => const AppScaffold(body: Text('shell page')),
          ),
          GoRoute(
            path: '/profile',
            builder: (_, _) => const Scaffold(body: Text('Profile route')),
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
            ChangeNotifierProvider.value(value: warmFocus()),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      // All four actions survive as BlockIconButtons with their tooltips;
      // no bare Material IconButton is left in the chrome.
      final buttons = tester
          .widgetList<BlockIconButton>(find.byType(BlockIconButton))
          .toList();
      expect(buttons, hasLength(4));
      expect(
        buttons.map((b) => b.tooltip),
        containsAll([
          'Switch language',
          'Switch to light mode',
          'Profile',
          'Logout',
        ]),
      );
      expect(find.byIcon(Icons.translate), findsOneWidget);
      expect(find.byIcon(Icons.light_mode_outlined), findsOneWidget);
      expect(find.byIcon(Icons.person_outline), findsOneWidget);
      expect(find.byIcon(Icons.logout), findsOneWidget);
      expect(find.byType(IconButton), findsNothing);

      // The cluster block: 2px strokeSubtle border, hard cast (4,4) shadow,
      // raised surface at 95% under fgSecondary icons (the frozen
      // "utility icon pill" WCAG pair).
      final cluster = tester.widget<Container>(
        find
            .ancestor(
              of: find.byType(BlockIconButton).first,
              matching: find.byType(Container),
            )
            .first,
      );
      final decoration = cluster.decoration! as BoxDecoration;
      final border = decoration.border! as Border;
      expect(border.top.width, 2);
      expect(border.top.color, const Color(0xFF7B8CE0));
      expect(
        decoration.boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(4, 4))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );
      expect(decoration.color!.a, closeTo(0.95, 0.001));

      // The profile action still routes.
      await tester.tap(find.byIcon(Icons.person_outline));
      await tester.pumpAndSettle();
      expect(find.text('Profile route'), findsOneWidget);
    },
  );
}
