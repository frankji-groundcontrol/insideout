import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/features/landing/landing_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

void main() {
  for (final dark in const [false, true]) {
    testWidgets(
      'auth wordmark sits inside the modal card at 1440x900 (${dark ? 'dark' : 'light'}, zh-CN)',
      (tester) async {
        await tester.binding.setSurfaceSize(const Size(1440, 900));
        addTearDown(() => tester.binding.setSurfaceSize(null));
        final session = Session()
          ..ready = true
          ..api = QuietAuthApi();
        await tester.pumpWidget(
          MultiProvider(
            providers: [
              ChangeNotifierProvider.value(value: session),
              ChangeNotifierProvider(
                create: (_) => Appearance(
                  store: MemorySettingsStore(),
                  locale: 'zh-CN',
                  dark: dark,
                ),
              ),
            ],
            child: MaterialApp(
              theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
              darkTheme: cobaltTheme(Cobalt.dark, brightness: Brightness.dark),
              themeMode: dark ? ThemeMode.dark : ThemeMode.light,
              builder: (context, child) {
                return MediaQuery(
                  data: MediaQuery.of(
                    context,
                  ).copyWith(disableAnimations: true),
                  child: child!,
                );
              },
              home: const LandingPage(initialPrompt: AuthPrompt.login),
            ),
          ),
        );
        await tester.pump();
        await tester.pump();

        expect(find.byKey(AuthDoor.wordmarkKey), findsOneWidget);
        expect(find.byKey(AuthDoor.cardKey), findsOneWidget);
        final wordmark = tester.getRect(find.byKey(AuthDoor.wordmarkKey));
        final card = tester.getRect(find.byKey(AuthDoor.cardKey));
        expect(wordmark.width, greaterThan(0));
        expect(wordmark.height, greaterThan(0));
        expect(wordmark.left, greaterThanOrEqualTo(card.left));
        expect(wordmark.top, greaterThanOrEqualTo(card.top));
        expect(wordmark.right, lessThanOrEqualTo(card.right));
        expect(wordmark.bottom, lessThanOrEqualTo(card.bottom));
      },
    );
  }
}
