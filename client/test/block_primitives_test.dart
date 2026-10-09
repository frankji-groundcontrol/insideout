import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/block_field.dart';
import 'package:insideout/theme/block_modal.dart';
import 'package:insideout/theme/block_state.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:insideout/widgets/pressable_block.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets(
    'BlockButton: rest cast (4,4), hover castSm + (2,2), active flush + (4,4)',
    (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        _app(
          Center(
            child: BlockButton(label: 'Go', onPressed: () => taps++),
          ),
        ),
      );

      // rest: full cast, no travel.
      expect(_shadow(tester).single.offset, const Offset(4, 4));
      expect(_travelX(tester), 0);

      final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
      await mouse.addPointer(location: Offset.zero);
      addTearDown(mouse.removePointer);
      await mouse.moveTo(tester.getCenter(find.byType(BlockButton)));
      await tester.pumpAndSettle();

      // hover: small cast, block travels (2,2) into it.
      expect(_shadow(tester).single.offset, const Offset(2, 2));
      expect(_travelX(tester), 2);
      expect(_travelY(tester), 2);

      await mouse.down(tester.getCenter(find.byType(BlockButton)));
      await tester.pumpAndSettle();

      // active: shadow gone, block pressed (4,4) into the page.
      expect(_shadow(tester), isEmpty);
      expect(_travelX(tester), 4);
      expect(_travelY(tester), 4);

      await mouse.up();
      await tester.pumpAndSettle();
      expect(taps, 1);
      expect(_shadow(tester).single.offset, const Offset(2, 2));
    },
  );

  testWidgets('BlockButtonSize pins fixed outer heights, border included', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        const Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            BlockButton(label: 'Small', size: BlockButtonSize.sm),
            BlockButton(label: 'Medium', size: BlockButtonSize.md),
            BlockButton(label: 'Large', size: BlockButtonSize.lg),
          ],
        ),
      ),
    );
    final buttons = find.byType(BlockButton);
    final heights = [
      for (var i = 0; i < 3; i++) tester.getSize(buttons.at(i)).height,
    ];
    expect(heights, [32, 40, 52]);
  });

  testWidgets(
    'variants: ink-edged primary and danger, flat ghost, both themes',
    (tester) async {
      for (final dark in [true, false]) {
        final palette = dark ? Cobalt.dark : Cobalt.light;
        await tester.pumpWidget(
          _app(
            dark: dark,
            Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  BlockButton(label: 'P', onPressed: () {}),
                  BlockButton(
                    label: 'S',
                    variant: BlockButtonVariant.secondary,
                    onPressed: () {},
                  ),
                  BlockButton(
                    label: 'D',
                    variant: BlockButtonVariant.danger,
                    onPressed: () {},
                  ),
                  BlockButton(
                    label: 'G',
                    variant: BlockButtonVariant.ghost,
                    onPressed: () {},
                  ),
                ],
              ),
            ),
          ),
        );
        // MaterialApp cross-fades themes; settle so Cobalt.of reads the new
        // brightness instead of the animated midpoint.
        await tester.pumpAndSettle();
        final decos = tester
            .widgetList<AnimatedContainer>(find.byType(AnimatedContainer))
            .map((w) => w.decoration! as BoxDecoration)
            .toList();
        expect(decos[0].color, palette.btn);
        expect(_sideColors(decos[0]), everyElement(Cobalt.lightInk));
        expect(decos[1].color, palette.surfaceRaised);
        expect(_sideColors(decos[1]), everyElement(palette.strokeSubtle));
        expect(decos[2].color, palette.brandHover);
        expect(_sideColors(decos[2]), everyElement(Cobalt.lightInk));
        expect(decos[3].border, isNull);
        expect(decos[3].boxShadow, isEmpty);
      }
    },
  );

  testWidgets('BlockLoader is a square block, never a circular spinner', (
    tester,
  ) async {
    await tester.pumpWidget(_app(const Center(child: BlockLoader(size: 24))));
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
      _app(
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

  testWidgets(
    'showBlockModal: header, 2px divider, scrollable body, footer bar',
    (tester) async {
      await tester.pumpWidget(
        _app(
          Builder(
            builder: (context) => Center(
              child: BlockButton(
                label: 'Open',
                onPressed: () => showBlockModal<void>(
                  context: context,
                  title: 'Modal title',
                  body: const Text('Modal body copy'),
                  footer: [
                    BlockButton(
                      label: 'Done',
                      variant: BlockButtonVariant.secondary,
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();

      expect(find.text('Modal title'), findsOneWidget);
      expect(find.byType(BlockIconButton), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);

      final divider = tester.widget<Divider>(find.byType(Divider));
      expect(divider.height, 2);
      expect(divider.thickness, 2);
      expect(divider.color, Cobalt.dark.strokeSubtle);

      expect(
        find.widgetWithText(SingleChildScrollView, 'Modal body copy'),
        findsOneWidget,
      );

      expect(find.text('Done'), findsOneWidget);
      final footerBar = tester
          .widgetList<Container>(
            find.ancestor(
              of: find.text('Done'),
              matching: find.byType(Container),
            ),
          )
          .map((w) => w.decoration)
          .whereType<BoxDecoration>()
          .firstWhere((d) => d.color == Cobalt.dark.surfaceSunken);
      expect(footerBar.color, Cobalt.dark.surfaceSunken);

      final panel = tester.getSize(
        find
            .ancestor(
              of: find.text('Modal title'),
              matching: find.byType(Material),
            )
            .first,
      );
      expect(panel.width, 512);
    },
  );

  testWidgets('showBlockConfirm resolves true on the danger confirm', (
    tester,
  ) async {
    bool? result;
    await tester.pumpWidget(
      _app(
        Builder(
          builder: (context) => Center(
            child: BlockButton(
              label: 'Ask',
              onPressed: () async {
                result = await showBlockConfirm(
                  context: context,
                  title: 'Delete node?',
                  message: 'This cannot be undone.',
                  danger: true,
                );
              },
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Ask'));
    await tester.pumpAndSettle();
    expect(find.text('Delete node?'), findsOneWidget);
    expect(find.text('This cannot be undone.'), findsOneWidget);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(result, isTrue);
    expect(find.text('Delete node?'), findsNothing);
  });

  testWidgets('BlockBanner paints its 4px tone border', (tester) async {
    await tester.pumpWidget(_app(const BlockBanner(message: 'Heads up')));
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
        _app(FieldCard(onTap: () => taps++, child: const Text('Tap me'))),
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
      await tester.pumpWidget(_app(const FieldCard(child: Text('Still'))));
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

  testWidgets(
    'BlockField: 44 tall, label above, focus flips the border to brand',
    (tester) async {
      await tester.pumpWidget(
        _app(
          const Center(
            child: SizedBox(
              width: 320,
              child: BlockField(label: 'Name', hint: 'Your name'),
            ),
          ),
        ),
      );
      expect(_fieldBoxSize(tester).height, Cobalt.controlHeight);
      expect(
        tester.getTopLeft(find.text('Name')).dy,
        lessThan(tester.getTopLeft(find.byType(TextField)).dy),
      );
      expect(_fieldBorder(tester), Cobalt.dark.strokeSubtle);
      await tester.tap(find.byType(TextField));
      // pump, not pumpAndSettle: a focused caret blinks forever.
      await tester.pump();
      expect(_fieldBorder(tester), Cobalt.dark.brand);
    },
  );

  testWidgets('BlockField: dense is 40 and the error paints fgDanger below', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        const Center(
          child: SizedBox(
            width: 320,
            child: BlockField(dense: true, errorText: 'Required'),
          ),
        ),
      ),
    );
    expect(_fieldBoxSize(tester).height, Cobalt.controlHeightDense);
    expect(_fieldBorder(tester), Cobalt.dark.fgDanger);
    final errorText = tester.widget<Text>(find.text('Required'));
    expect(errorText.style!.fontSize, 12);
    expect(errorText.style!.color, Cobalt.dark.fgDanger);
  });

  testWidgets('BlockSelect: 44-tall raised box, picks an option', (
    tester,
  ) async {
    String? picked;
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (context, setState) => Center(
            child: SizedBox(
              width: 320,
              child: BlockSelect<String>(
                label: 'Status',
                value: 'a',
                options: const [
                  (value: 'a', label: 'Alpha'),
                  (value: 'b', label: 'Beta'),
                ],
                onChanged: (v) => setState(() => picked = v),
              ),
            ),
          ),
        ),
      ),
    );
    final box = find
        .ancestor(
          of: find.byType(DropdownButton<String>),
          matching: find.byType(Container),
        )
        .first;
    expect(tester.getSize(box).height, Cobalt.controlHeight);
    await tester.tap(find.byType(DropdownButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Beta').last);
    await tester.pumpAndSettle();
    expect(picked, 'b');
  });

  testWidgets('BlockError localizes into the banner and offers a retry', (
    tester,
  ) async {
    var retries = 0;
    await tester.pumpWidget(
      _app(
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

Widget _app(Widget child, {bool dark = true}) {
  final palette = dark ? Cobalt.dark : Cobalt.light;
  return ChangeNotifierProvider(
    create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
    child: MaterialApp(
      theme: cobaltTheme(
        palette,
        brightness: dark ? Brightness.dark : Brightness.light,
      ),
      home: Scaffold(body: child),
    ),
  );
}

BoxDecoration _deco(WidgetTester tester) =>
    tester
            .widget<AnimatedContainer>(
              find.descendant(
                of: find.byType(BlockButton),
                matching: find.byType(AnimatedContainer),
              ),
            )
            .decoration!
        as BoxDecoration;

List<BoxShadow> _shadow(WidgetTester tester) => _deco(tester).boxShadow!;

double _travelX(WidgetTester tester) => _transform(tester)!.getTranslation().x;

double _travelY(WidgetTester tester) => _transform(tester)!.getTranslation().y;

Matrix4? _transform(WidgetTester tester) => tester
    .widget<AnimatedContainer>(
      find.descendant(
        of: find.byType(BlockButton),
        matching: find.byType(AnimatedContainer),
      ),
    )
    .transform;

List<Color> _sideColors(BoxDecoration d) {
  final b = d.border! as Border;
  return [b.top.color, b.right.color, b.bottom.color, b.left.color];
}

double _loaderOpacity(WidgetTester tester) => tester
    .widget<Opacity>(
      find.descendant(
        of: find.byType(BlockLoader),
        matching: find.byType(Opacity),
      ),
    )
    .opacity;

Finder _fieldBox() => find
    .ancestor(of: find.byType(TextField), matching: find.byType(Container))
    .first;

Size _fieldBoxSize(WidgetTester tester) => tester.getSize(_fieldBox());

Color _fieldBorder(WidgetTester tester) {
  final deco =
      tester.widget<Container>(_fieldBox()).decoration! as BoxDecoration;
  return (deco.border! as Border).top.color;
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
