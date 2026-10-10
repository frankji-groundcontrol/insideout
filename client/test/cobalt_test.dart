import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_chip.dart';
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
    expect(Cobalt.light.strokeSubtle, const Color(0xFF7B8CE0));
    expect(Cobalt.light.strokeStrong, const Color(0xFF5A6FE0));

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
    expect(Cobalt.dark.surfaceSunken, const Color(0xFF16181E));
    expect(Cobalt.dark.surfaceRaised, const Color(0xFF14171D));
    expect(Cobalt.dark.statusNeutralBg, const Color(0xFF1A1F2E));
    expect(Cobalt.dark.brandSubtle, Cobalt.hue950);
    expect(Cobalt.dark.surfaceSunken, isNot(Cobalt.hue950));
    expect(Cobalt.dark.btn, Cobalt.hue500);

    final theme = cobaltTheme(Cobalt.dark, brightness: Brightness.dark);
    expect(theme.scaffoldBackgroundColor, Cobalt.darkGround);
    expect(theme.colorScheme.primary, Cobalt.hue500);
  });

  test('cobalt-neo decree: every radius token is squared to 0', () {
    expect(Cobalt.radiusControl, 0.0);
    expect(Cobalt.radiusCard, 0.0);
    expect(Cobalt.radiusHero, 0.0);
    expect(Cobalt.radiusPill, 0.0);
  });

  test(
    'cobalt-neo decree: border widths and press-the-block motion tokens',
    () {
      expect(Cobalt.borderContainer, 2.0);
      expect(Cobalt.borderDivider, 1.0);
      expect(Cobalt.pressDuration, const Duration(milliseconds: 100));
      expect(Cobalt.hoverOffset, const Offset(2, 2));
      expect(Cobalt.activeOffset, const Offset(4, 4));
    },
  );

  test('spacing decree: the named space tokens are the 4px grid', () {
    expect(Cobalt.space1, 4.0);
    expect(Cobalt.space2, 8.0);
    expect(Cobalt.space3, 12.0);
    expect(Cobalt.space4, 16.0);
    expect(Cobalt.space6, 24.0);
    expect(Cobalt.space8, 32.0);
    expect(Cobalt.space10, 40.0);
  });

  test('cast shadows: hard offsets, zero blur and spread, per-theme ink', () {
    expect(castShadowColor(Brightness.dark), const Color(0xFF05070A));
    expect(castShadowColor(Brightness.light), const Color(0x26000000));
    expect(cast(Brightness.dark).single.offset, const Offset(4, 4));
    expect(castSm(Brightness.dark).single.offset, const Offset(2, 2));
    expect(cast(Brightness.light).single.color, const Color(0x26000000));
    for (final shadows in [
      cast(Brightness.light),
      castSm(Brightness.light),
      cast(Brightness.dark),
      castSm(Brightness.dark),
    ]) {
      for (final s in shadows) {
        expect(s.blurRadius, 0.0);
        expect(s.spreadRadius, 0.0);
      }
    }
  });

  test(
    'cast shadows darken their own ground on both themes (decorative shadows, WCAG 1.4.11 exempt)',
    () {
      // A shadow must read as a shadow: strictly darker than the surface it
      // lands on. A 3:1 ratio against its own ground is neither possible for
      // real cast ink nor required — 1.4.11 covers informational graphics only.
      for (final s in [
        Cobalt.dark.surfaceBase,
        Cobalt.dark.surfaceRaised,
        Cobalt.dark.surfaceSunken,
        Cobalt.dark.statusNeutralBg,
      ]) {
        expect(
          _luminance(castShadowColor(Brightness.dark)),
          lessThan(_luminance(s)),
        );
      }
      for (final s in [
        Cobalt.light.surfaceBase,
        Cobalt.light.surfaceRaised,
        Cobalt.light.surfaceSunken,
      ]) {
        final blended = _over(const Color(0xFF000000), s, 0x26 / 0xFF);
        expect(_luminance(blended), lessThan(_luminance(s)));
      }
    },
  );

  test('neutral dark ramp: strokes clear 3:1 on every dark surface', () {
    expect(Cobalt.dark.strokeSubtle, const Color(0xFF646A78));
    expect(Cobalt.dark.strokeStrong, const Color(0xFF788098));
    for (final g in [
      Cobalt.dark.surfaceBase,
      Cobalt.dark.surfaceRaised,
      Cobalt.dark.surfaceSunken,
      Cobalt.dark.statusNeutralBg,
    ]) {
      expect(_contrast(Cobalt.dark.strokeSubtle, g), greaterThanOrEqualTo(3.0));
      expect(_contrast(Cobalt.dark.strokeStrong, g), greaterThanOrEqualTo(3.0));
    }
  });

  test('dock warn amber clears 3:1 on the dock brand background', () {
    const amber = Color(0xFFFFD166);
    expect(_contrast(amber, Cobalt.light.brand), greaterThanOrEqualTo(3.0));
    expect(
      _contrast(amber, Cobalt.light.brandHover),
      greaterThanOrEqualTo(3.0),
    );
    expect(_contrast(amber, Cobalt.dark.brand), greaterThanOrEqualTo(3.0));
  });

  test('chip text is cobalt-700 on the hue50 chip background (4.5:1 text)', () {
    expect(Cobalt.light.statusNeutralFg, Cobalt.hue700);
    expect(
      _contrast(Cobalt.light.statusNeutralFg, Cobalt.hue50),
      greaterThanOrEqualTo(4.5),
    );
    expect(
      _contrast(Cobalt.dark.statusNeutralFg, Cobalt.dark.statusNeutralBg),
      greaterThanOrEqualTo(4.5),
    );
  });

  // The chip label is 12px w600, i.e. normal text: every tone owes 4.5:1. The
  // light danger chip (cobalt-700 on cobalt-200) measured 4.49 and nothing
  // pinned it, because the test above only reads the neutral pair.
  testWidgets(
    'every chip tone: the label clears 4.5:1 on its fill in both themes',
    (tester) async {
      for (final (palette, brightness) in [
        (Cobalt.light, Brightness.light),
        (Cobalt.dark, Brightness.dark),
      ]) {
        for (final tone in ShadeTone.values) {
          await tester.pumpWidget(
            MaterialApp(
              theme: cobaltTheme(palette, brightness: brightness),
              home: Scaffold(
                body: Center(
                  child: ShadeChip(label: 'x', tone: tone),
                ),
              ),
            ),
          );
          final chip = tester.widget<Container>(
            find.descendant(
              of: find.byType(ShadeChip),
              matching: find.byType(Container),
            ),
          );
          final fill = (chip.decoration! as BoxDecoration).color!;
          final label = tester.widget<Text>(find.text('x')).style!.color!;
          expect(
            _contrast(label, fill),
            greaterThanOrEqualTo(4.5),
            reason: '${tone.name} chip in ${brightness.name}',
          );
        }
      }
    },
  );

  test(
    'theme wiring: squared shapes, container borders, flat surfaces, 100ms press motion',
    () {
      for (final entry in [
        (Cobalt.light, Brightness.light),
        (Cobalt.dark, Brightness.dark),
      ]) {
        final theme = cobaltTheme(entry.$1, brightness: entry.$2);

        final cardShape = theme.cardTheme.shape! as RoundedRectangleBorder;
        expect(cardShape.borderRadius, BorderRadius.zero);
        expect(cardShape.side.width, Cobalt.borderContainer);
        expect(theme.cardTheme.elevation, 0);

        final dialogShape = theme.dialogTheme.shape! as RoundedRectangleBorder;
        expect(dialogShape.borderRadius, BorderRadius.zero);
        expect(dialogShape.side.width, Cobalt.borderContainer);
        expect(theme.dialogTheme.elevation, 0);

        final snackShape = theme.snackBarTheme.shape! as RoundedRectangleBorder;
        expect(snackShape.borderRadius, BorderRadius.zero);
        expect(snackShape.side.width, Cobalt.borderContainer);
        expect(theme.snackBarTheme.elevation, 0);

        final inputBorder =
            theme.inputDecorationTheme.enabledBorder! as OutlineInputBorder;
        expect(inputBorder.borderRadius, BorderRadius.zero);
        expect(inputBorder.borderSide.width, Cobalt.borderContainer);

        final chipShape = theme.chipTheme.shape! as RoundedRectangleBorder;
        expect(chipShape.borderRadius, BorderRadius.zero);
        expect(theme.chipTheme.side?.width, Cobalt.borderContainer);

        expect(
          theme.filledButtonTheme.style?.animationDuration,
          Cobalt.pressDuration,
        );
        expect(
          theme.elevatedButtonTheme.style?.animationDuration,
          Cobalt.pressDuration,
        );
        expect(
          theme.outlinedButtonTheme.style?.animationDuration,
          Cobalt.pressDuration,
        );
        expect(
          theme.textButtonTheme.style?.animationDuration,
          Cobalt.pressDuration,
        );
      }
    },
  );

  test('no celadon, vermilion, or ink-as-primary leftovers', () {
    expect(Cobalt.light.surfaceBase, isNot(const Color(0xFFE4EAE4)));
    expect(Cobalt.dark.surfaceBase, isNot(const Color(0xFF161A17)));
    expect(Cobalt.light.btn, isNot(const Color(0xFF211E1A)));
    expect(Cobalt.light.brand, isNot(const Color(0xFFC8402F)));
    expect(Cobalt.dark.brand, isNot(const Color(0xFFD84A31)));
  });

  test(
    'UI sans is Archivo + Noto Sans SC; coach serif is Source Serif 4 + Noto Serif SC',
    () {
      expect(Cobalt.sansWeb, 'Archivo');
      expect(Cobalt.sansCjkWeb, 'Noto Sans SC');
      expect(Cobalt.serifWeb, 'Source Serif 4');
      expect(Cobalt.serifCjkWeb, 'Noto Serif SC');
      expect(Cobalt.sansNative, 'NotoSansSC');
      expect(Cobalt.serifNative, 'NotoSerifSC');
    },
  );

  test('ideas wall display clamps 64–320 from 10vw', () {
    expect(Cobalt.ideasDisplaySize(400), 64);
    expect(Cobalt.ideasDisplaySize(1200), 120);
    expect(Cobalt.ideasDisplaySize(4000), 320);
  });

  test(
    'utility icon pill: fgSecondary icons on surfaceRaised/95% clear 3:1 (WCAG relative luminance)',
    () {
      for (final c in [Cobalt.light, Cobalt.dark]) {
        final pill = _over(c.surfaceRaised, Cobalt.hue950, 0.95);
        expect(_contrast(c.fgSecondary, pill), greaterThanOrEqualTo(3.0));
      }
    },
  );
}

double _channel(double v) =>
    v <= 0.03928 ? v / 12.92 : math.pow((v + 0.055) / 1.055, 2.4).toDouble();

double _luminance(Color c) =>
    0.2126 * _channel(c.r) + 0.7152 * _channel(c.g) + 0.0722 * _channel(c.b);

Color _over(Color fg, Color bg, double alpha) => Color.fromARGB(
  255,
  ((fg.r * alpha + bg.r * (1 - alpha)) * 255.0).round(),
  ((fg.g * alpha + bg.g * (1 - alpha)) * 255.0).round(),
  ((fg.b * alpha + bg.b * (1 - alpha)) * 255.0).round(),
);

double _contrast(Color a, Color b) {
  final la = _luminance(a);
  final lb = _luminance(b);
  final hi = math.max(la, lb);
  final lo = math.min(la, lb);
  return (hi + 0.05) / (lo + 0.05);
}
