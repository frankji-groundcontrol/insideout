import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';

void main() {
  test('light cobalt: one hue, neutrals carry the page, primary is cobalt', () {
    expect(Cobalt.hue50, const Color(0xFFEEF2FF));
    expect(Cobalt.hue200, const Color(0xFFC7D2FE));
    expect(Cobalt.hue500, const Color(0xFF3B5BFF));
    expect(Cobalt.hue700, const Color(0xFF1D4ED8));
    expect(Cobalt.hue950, const Color(0xFF1E2A5A));
    expect(Cobalt.lightGround, const Color(0xFFF7F8FA));
    expect(Cobalt.lightInk, const Color(0xFF0B0D12));

    expect(Cobalt.light.surfaceBase, Cobalt.lightGround);
    expect(Cobalt.light.fgPrimary, Cobalt.lightInk);
    expect(Cobalt.light.brand, Cobalt.hue500);
    expect(Cobalt.light.btn, Cobalt.hue500);
    expect(Cobalt.light.brandSubtle, Cobalt.hue50);
    expect(Cobalt.light.strokeSubtle, Cobalt.hue200);

    final theme = cobaltTheme(Cobalt.light, brightness: Brightness.light);
    expect(theme.scaffoldBackgroundColor, Cobalt.lightGround);
    expect(theme.colorScheme.primary, Cobalt.hue500);
    expect(theme.colorScheme.secondary, isNot(const Color(0xFFC8402F)));
    expect(theme.textTheme.bodyLarge?.fontSize, 17.5);
    expect(theme.textTheme.titleLarge!.fontSize! >= 26, isTrue);
    expect(theme.textTheme.displaySmall!.fontSize! >= 48, isTrue);
    expect(theme.textTheme.displaySmall!.fontSize! <= 58, isTrue);
    expect(theme.textTheme.bodySmall?.fontSize, 12);
  });

  test('dark cobalt: field and type invert, primary stays cobalt', () {
    expect(Cobalt.darkGround, const Color(0xFF0B0D12));
    expect(Cobalt.darkType, const Color(0xFFF7F8FA));
    expect(Cobalt.dark.surfaceBase, Cobalt.darkGround);
    expect(Cobalt.dark.fgPrimary, Cobalt.darkType);
    expect(Cobalt.dark.surfaceSunken, Cobalt.hue950);
    expect(Cobalt.dark.btn, Cobalt.hue500);

    final theme = cobaltTheme(Cobalt.dark, brightness: Brightness.dark);
    expect(theme.scaffoldBackgroundColor, Cobalt.darkGround);
    expect(theme.colorScheme.primary, Cobalt.hue500);
  });

  test('no celadon, vermilion, or ink-as-primary leftovers', () {
    expect(Cobalt.light.surfaceBase, isNot(const Color(0xFFE4EAE4)));
    expect(Cobalt.dark.surfaceBase, isNot(const Color(0xFF161A17)));
    expect(Cobalt.light.btn, isNot(const Color(0xFF211E1A)));
    expect(Cobalt.light.brand, isNot(const Color(0xFFC8402F)));
    expect(Cobalt.dark.brand, isNot(const Color(0xFFD84A31)));
  });

  test('UI sans is Archivo + Noto Sans SC; coach serif is Source Serif 4 + Noto Serif SC', () {
    expect(Cobalt.sansWeb, 'Archivo');
    expect(Cobalt.sansCjkWeb, 'Noto Sans SC');
    expect(Cobalt.serifWeb, 'Source Serif 4');
    expect(Cobalt.serifCjkWeb, 'Noto Serif SC');
    expect(Cobalt.sansNative, 'NotoSansSC');
    expect(Cobalt.serifNative, 'NotoSerifSC');
  });

  test('ideas wall display clamps 64–320 from 10vw', () {
    expect(Cobalt.ideasDisplaySize(400), 64);
    expect(Cobalt.ideasDisplaySize(1200), 120);
    expect(Cobalt.ideasDisplaySize(4000), 320);
  });
}
