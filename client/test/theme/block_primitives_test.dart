import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/block_state.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:insideout/widgets/pressable_block.dart';

import '_block_test_helpers.dart';

void main() {
  testWidgets('BlockLoader is a square block, never a circular spinner', (
    tester,
  ) async {
    await tester.pumpWidget(host(const Center(child: BlockLoader(size: 24))));
    expect(tester.getSize(find.byType(BlockLoader)), const Size(24, 24));
    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(_loaderOpacity(tester), 0.25);
    await tester.pump(const Duration(milliseconds: 450));
    expect(_loaderOpacity(tester), greaterThan(0.25));
    expect(_loaderOpacity(tester), lessThan(1.0));
  });

  testWidgets('BlockLoader goes static under MediaQuery.disableAnimationsOf', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(
        Builder(
          builder: (context) => MediaQuery(
            data: MediaQuery.of(context).copyWith(disableAnimations: true),
            child: const Center(child: BlockLoader(size: 24)),
          ),
        ),
      ),
    );
    expect(_loaderOpacity(tester), 1.0);
    await tester.pump(const Duration(milliseconds: 450));
    expect(_loaderOpacity(tester), 1.0);
    await tester.pump(const Duration(milliseconds: 900));
    expect(_loaderOpacity(tester), 1.0);
  });

  testWidgets('BlockBanner paints its 4px tone border', (tester) async {
    await tester.pumpWidget(host(const BlockBanner(message: 'Heads up')));
    final container = tester.widget<Container>(
      find
          .ancestor(of: find.text('Heads up'), matching: find.byType(Container))
          .first,
    );
    final deco = container.decoration! as BoxDecoration;
    final border = deco.border! as Border;
    expect(border.left.width, 4);
    expect(border.left.color, Cobalt.dark.fgDanger);
    expect(border.top, BorderSide.none);
    expect(deco.color, Cobalt.dark.surfaceSunken);
    expect(tester.getSize(find.byType(BlockBanner)).width, 800);
  });

  testWidgets(
    'FieldCard with onTap drops the InkWell for one PressableBlock cast',
    (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        host(FieldCard(onTap: () => taps++, child: const Text('Tap me'))),
      );
      expect(find.byType(InkWell), findsNothing);
      expect(
        find.descendant(
          of: find.byType(FieldCard),
          matching: find.byType(PressableBlock),
        ),
        findsOneWidget,
      );
      final castBoxes = tester
          .widgetList<AnimatedContainer>(
            find.descendant(
              of: find.byType(FieldCard),
              matching: find.byType(AnimatedContainer),
            ),
          )
          .where(
            (w) =>
                (w.decoration as BoxDecoration?)?.boxShadow?.isNotEmpty ??
                false,
          );
      expect(castBoxes, hasLength(1));
      // The lone cast sits at rest (4,4); the FieldCard no longer paints its
      // own shadow under the PressableBlock (the InkWell path used to double
      // it). AnimatedContainer renders one inner Container per shadow, so a
      // single cast source here proves there is no second one.
      expect(
        (castBoxes.single.decoration! as BoxDecoration)
            .boxShadow!
            .single
            .offset,
        const Offset(4, 4),
      );
      await tester.tap(find.text('Tap me'));
      expect(taps, 1);
    },
  );

  testWidgets(
    'FieldCard without onTap keeps the static cast and antiAlias clip',
    (tester) async {
      await tester.pumpWidget(host(const FieldCard(child: Text('Still'))));
      expect(find.byType(PressableBlock), findsNothing);
      expect(find.byType(InkWell), findsNothing);
      final container = tester.widget<Container>(
        find
            .descendant(
              of: find.byType(FieldCard),
              matching: find.byType(Container),
            )
            .first,
      );
      final deco = container.decoration! as BoxDecoration;
      expect(deco.boxShadow!.single.offset, const Offset(4, 4));
      final material = tester.widget<Material>(
        find.descendant(
          of: find.byType(FieldCard),
          matching: find.byType(Material),
        ),
      );
      expect(material.clipBehavior, Clip.antiAlias);
    },
  );

  testWidgets('BlockError localizes into the banner and offers a retry', (
    tester,
  ) async {
    var retries = 0;
    await tester.pumpWidget(
      host(
        BlockError(error: TimeoutException('boom'), onRetry: () => retries++),
      ),
    );
    expect(find.text('Something went wrong'), findsOneWidget);
    expect(find.text('Network problem — check the connection'), findsOneWidget);
    await tester.tap(find.text('Retry'));
    expect(retries, 1);
  });

  test(
    'WCAG: the lightInk primary border clears 3:1 non-text against cobalt-500',
    () {
      expect(Cobalt.light.btn, Cobalt.hue500);
      expect(Cobalt.dark.btn, Cobalt.hue500);
      expect(
        _contrast(Cobalt.lightInk, Cobalt.hue500),
        greaterThanOrEqualTo(3.0),
      );
    },
  );

  test('WCAG: every block tone clears 3:1 accent and 4.5:1 ink', () {
    for (final (palette, brightness) in [
      (Cobalt.light, Brightness.light),
      (Cobalt.dark, Brightness.dark),
    ]) {
      for (final tone in BlockTone.values) {
        final accent = blockToneAccent(palette, brightness, tone);
        final ink = blockToneInk(palette, brightness, tone);
        expect(
          _contrast(accent, palette.surfaceSunken),
          greaterThanOrEqualTo(3.0),
          reason: '$tone accent on ${brightness.name} ground',
        );
        expect(
          _contrast(ink, _over(accent, palette.surfaceRaised, 0.15)),
          greaterThanOrEqualTo(4.5),
          reason: '$tone ink on tinted ${brightness.name} badge',
        );
      }
    }
  });
}

double _loaderOpacity(WidgetTester tester) => tester
    .widget<Opacity>(
      find.descendant(
        of: find.byType(BlockLoader),
        matching: find.byType(Opacity),
      ),
    )
    .opacity;

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
