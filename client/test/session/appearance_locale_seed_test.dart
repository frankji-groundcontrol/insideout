import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/session/appearance.dart';

// First visit: no locale is stored yet, so hydration must honor the
// browser's languages (en* -> en-US, anything else falls back to zh-CN).
// A stored choice stays authoritative — seeding only fills the silence.
void main() {
  testWidgets('an English browser and no stored choice resolve en-US', (
    tester,
  ) async {
    tester.platformDispatcher.localesTestValue = const [Locale('en', 'US')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);

    final appearance = Appearance(store: MemorySettingsStore());
    await appearance.hydrate();

    expect(appearance.locale, 'en-US');
  });

  testWidgets('any English variant browser resolves en-US', (tester) async {
    tester.platformDispatcher.localesTestValue = const [Locale('en', 'GB')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);

    final appearance = Appearance(store: MemorySettingsStore());
    await appearance.hydrate();

    expect(appearance.locale, 'en-US');
  });

  testWidgets('a browser in neither language falls back to zh-CN', (
    tester,
  ) async {
    tester.platformDispatcher.localesTestValue = const [Locale('fr', 'FR')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);

    final appearance = Appearance(store: MemorySettingsStore());
    await appearance.hydrate();

    expect(appearance.locale, 'zh-CN');
  });

  testWidgets('a stored choice beats the browser locale', (tester) async {
    tester.platformDispatcher.localesTestValue = const [Locale('en', 'US')];
    addTearDown(tester.platformDispatcher.clearLocalesTestValue);
    final store = MemorySettingsStore()..values[Appearance.localeKey] = 'zh-CN';

    final appearance = Appearance(store: store);
    await appearance.hydrate();

    expect(appearance.locale, 'zh-CN');
  });
}
