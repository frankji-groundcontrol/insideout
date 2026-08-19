import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/session/appearance.dart';

void main() {
  test('starts on Nuxt default locale and light theme', () {
    final appearance = Appearance(store: MemorySettingsStore());
    expect(appearance.locale, 'zh-CN');
    expect(appearance.dark, isFalse);
    expect(appearance.t('nav.dashboard'), '工作台');
  });

  test('toggle locale and theme persist through hydrate', () async {
    final store = MemorySettingsStore();
    final first = Appearance(store: store);
    await first.toggleLocale();
    await first.toggleTheme();
    expect(first.locale, 'en-US');
    expect(first.dark, isTrue);
    expect(first.t('nav.dashboard'), 'Dashboard');

    final restored = Appearance(store: store);
    await restored.hydrate();
    expect(restored.locale, 'en-US');
    expect(restored.dark, isTrue);
  });
}
