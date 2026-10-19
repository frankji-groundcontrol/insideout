import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/features/landing/landing_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

// rv4-PUB-02: the hosted service keeps an account, projects and coach
// conversations, and the sign-up screen said nothing about it. The notice is
// PRIVACY.md at the repository root (the web build does not bundle it); the
// sign-up screen links to that copy. A tap copies the address: the client has
// no URL launcher, so the link does not open a tab.

Future<void> _mount(
  WidgetTester tester, {
  required String locale,
  required AuthPrompt prompt,
}) async {
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
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
      ],
      child: MaterialApp(
        theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(disableAnimations: true),
          child: child!,
        ),
        home: LandingPage(initialPrompt: prompt),
      ),
    ),
  );
  await tester.pump();
  await tester.pump();
}

void _expectInsideCard(WidgetTester tester, Finder link) {
  final card = tester.getRect(find.byKey(AuthDoor.cardKey));
  final rect = tester.getRect(link);
  expect(
    card.contains(rect.topLeft) && card.contains(rect.bottomRight),
    isTrue,
    reason: 'the link $rect must sit inside the modal card $card',
  );
}

void main() {
  for (final (locale, label) in const [
    ('en-US', 'Privacy notice'),
    ('zh-CN', '隐私声明'),
  ]) {
    for (final prompt in AuthPrompt.values) {
      testWidgets('the ${prompt.name} prompt links to the privacy notice '
          '($locale)', (tester) async {
        await _mount(tester, locale: locale, prompt: prompt);

        final link = find.widgetWithText(BlockButton, label);
        expect(link, findsOneWidget);
        _expectInsideCard(tester, link);
      });
    }
  }

  testWidgets('a tap copies the notice address and says so', (tester) async {
    String? copied;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          copied = (call.arguments as Map)['text'] as String?;
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    await _mount(tester, locale: 'en-US', prompt: AuthPrompt.register);

    await tester.tap(find.widgetWithText(BlockButton, 'Privacy notice'));
    await tester.pump();

    final address = Uri.parse(copied ?? '');
    expect(address.scheme, 'https');
    expect(address.path, endsWith('/PRIVACY.md'));
    expect(find.textContaining('Link copied'), findsOneWidget);
  });

  test('the notice address is one named constant in the auth door', () {
    final source = File('lib/features/auth/auth_door.dart').readAsStringSync();
    expect(
      RegExp(
        r"const privacyNoticeUrl\s*=\s*'https://[^']+/PRIVACY\.md';",
      ).hasMatch(source),
      isTrue,
      reason: 'auth_door.dart must declare `const privacyNoticeUrl`',
    );
  });

  test('PRIVACY.md at the repository root says what the link promises', () {
    final file = File('../PRIVACY.md');
    expect(file.existsSync(), isTrue, reason: 'the link points at this file');
    final text = file.readAsStringSync();
    for (final heading in const [
      '## What is stored',
      '## Who else receives data',
      '## Retention',
      '## Contact',
    ]) {
      expect(text, contains(heading));
    }
    // The processors the code can send user data to, named, not summarised.
    for (final processor in const ['model provider', 'Resend', 'GitHub']) {
      expect(text, contains(processor));
    }
  });
}
