import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../l10n/l10n.dart';

abstract class SettingsStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
}

class MemorySettingsStore implements SettingsStore {
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    values[key] = value;
  }
}

class SecureSettingsStore implements SettingsStore {
  SecureSettingsStore([FlutterSecureStorage? storage]) : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) => _storage.write(key: key, value: value);
}

class Appearance extends ChangeNotifier {
  Appearance({SettingsStore? store, this.locale = defaultLocale, this.dark = false})
      : _store = store ?? MemorySettingsStore();

  static const localeKey = 'insideout_locale';
  static const themeKey = 'insideout_theme';

  final SettingsStore _store;
  String locale;
  bool dark;

  ThemeMode get themeMode => dark ? ThemeMode.dark : ThemeMode.light;

  String t(String key, [Map<String, Object?>? args]) => lookup(locale, key, args);

  List<String> suggestions(String stage) => coachSuggestions(locale, stage);

  Future<void> hydrate() async {
    final savedLocale = await _store.read(localeKey);
    final savedTheme = await _store.read(themeKey);
    if (savedLocale == 'en-US' || savedLocale == 'zh-CN') locale = savedLocale!;
    if (savedTheme == 'dark') dark = true;
    if (savedTheme == 'light') dark = false;
    notifyListeners();
  }

  Future<void> toggleLocale() async {
    locale = locale == 'zh-CN' ? 'en-US' : 'zh-CN';
    await _store.write(localeKey, locale);
    notifyListeners();
  }

  Future<void> toggleTheme() async {
    dark = !dark;
    await _store.write(themeKey, dark ? 'dark' : 'light');
    notifyListeners();
  }
}
