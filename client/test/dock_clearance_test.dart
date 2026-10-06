import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/app.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:provider/provider.dart';

/// Wave 1.1 / 6.1: the dock is an [Align] at the bottom of [AppScaffold].
/// Replacing that with an expanding [Center] lifts the pill into the
/// viewport and covers the body. This test fails in that case.
void main() {
  testWidgets(
    'at 1440x900 the dock stays on the bottom and clears the landmark',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1440, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final session = Session()
        ..ready = true
        ..user = UserProfile(
          id: 'u1',
          email: 'demo@insideout.local',
          username: 'demo',
          bio: '',
          keywords: const [],
        );
      final focus = LoopFocus();
      final router = GoRouter(
        initialLocation: '/dashboard',
        routes: [
          GoRoute(
            path: '/dashboard',
            builder: (_, _) => AppScaffold(
              body: Padding(
                padding: const EdgeInsets.only(bottom: ProductDock.height + 16),
                child: const Align(
                  alignment: Alignment.bottomCenter,
                  child: SizedBox(
                    key: Key('cleared-landmark'),
                    width: 120,
                    height: 24,
                    child: Text('cleared'),
                  ),
                ),
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
            ChangeNotifierProvider.value(value: focus),
          ],
          child: MaterialApp.router(
            theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
            routerConfig: router,
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      final dock = tester.getRect(find.byType(ProductDock));
      final landmark = tester.getRect(
        find.byKey(const Key('cleared-landmark')),
      );

      // Bottom band of a 900px surface. A mid-viewport Center puts top near 400.
      expect(dock.bottom, greaterThan(860));
      expect(dock.top, greaterThan(700));
      expect(dock.overlaps(landmark), isFalse);
      expect(landmark.bottom, lessThanOrEqualTo(dock.top));
    },
  );
}
