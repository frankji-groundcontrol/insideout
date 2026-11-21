import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/cobalt.dart';

import '_block_test_helpers.dart';

void main() {
  testWidgets(
    'BlockButton: rest cast (4,4), hover castSm + (2,2), active flush + (4,4)',
    (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        host(
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
      host(
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
          host(
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
