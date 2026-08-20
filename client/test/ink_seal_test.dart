import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/ink_seal.dart';
import 'package:insideout/theme/ink_seal_theme.dart';

void main() {
  test('light seal is vermilion, primary button is ink', () {
    expect(InkSeal.light.seal, const Color(0xFFC8402F));
    expect(InkSeal.light.btn, const Color(0xFF211E1A));
    expect(InkSeal.light.surfaceBase, const Color(0xFFE4EAE4));
    final theme = inkSealTheme(InkSeal.light, brightness: Brightness.light);
    expect(theme.scaffoldBackgroundColor, InkSeal.light.surfaceBase);
    expect(theme.colorScheme.primary, InkSeal.light.btn);
    expect(theme.colorScheme.secondary, InkSeal.light.seal);
  });

  test('dark ground is ink-night and the button inverts to paper', () {
    expect(InkSeal.dark.surfaceBase, const Color(0xFF161A17));
    expect(InkSeal.dark.btn, const Color(0xFFE6E9E1));
    final theme = inkSealTheme(InkSeal.dark, brightness: Brightness.dark);
    expect(theme.colorScheme.primary, InkSeal.dark.btn);
  });
}
