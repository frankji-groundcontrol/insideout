import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/widgets/pressable_block.dart';

import '_dock_test_helpers.dart';

void main() {
  testWidgets(
    'at 390px the dock collapses to icons: six items fit, no overflow, still tappable',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(390, 844));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // The unfinished long label is exactly the case that used to blow the
      // dock out to the viewport edges and shift sibling hit targets.
      final focus = warmFocus(committed: false);
      await pumpDock(tester, focus: focus);

      expect(tester.takeException(), isNull);

      // The unfinished label must not stretch the pill out to the 390px edges.
      final pill = tester.getRect(
        find.byWidgetPredicate(
          (w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration! as BoxDecoration).color == const Color(0xFF3B5BFF),
        ),
      );
      expect(pill.width, lessThan(360));

      final itemFinder = find.byType(PressableBlock);
      expect(itemFinder, findsNWidgets(6));

      // Every item is fully on the 390px surface, non-overlapping, and
      // comfortably tappable.
      final rects = <Rect>[
        for (var i = 0; i < 6; i++) tester.getRect(itemFinder.at(i)),
      ]..sort((a, b) => a.left.compareTo(b.left));
      for (var i = 0; i < rects.length; i++) {
        expect(rects[i].left, greaterThanOrEqualTo(0));
        expect(rects[i].right, lessThanOrEqualTo(390));
        expect(rects[i].width, greaterThanOrEqualTo(40));
        if (i > 0) {
          expect(rects[i].left, greaterThanOrEqualTo(rects[i - 1].right));
        }
      }

      // Icons only: no item labels, and the unfinished amber square remains.
      expect(find.text('Commit'), findsNothing);
      expect(find.text('Unfinished: Commit'), findsNothing);
      expect(
        find.byKey(const ValueKey('dock-unfinished-commit')),
        findsOneWidget,
      );

      // Icon-only items still navigate.
      await tester.tap(find.byIcon(Icons.description_outlined));
      await tester.pumpAndSettle();
      expect(find.text('Commit prd-1'), findsOneWidget);
    },
  );

  testWidgets(
    'dock items carry the cast at rest and press flat into the page over 100ms',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await pumpDock(tester, focus: focus);

      final item = find.byType(PressableBlock).first;
      final animated = find.descendant(
        of: item,
        matching: find.byType(AnimatedContainer),
      );
      final transform = find.descendant(
        of: item,
        matching: find.byType(Transform),
      );
      double translationX() =>
          tester.widget<Transform>(transform).transform.getTranslation().x;
      BoxDecoration shadowDecoration() =>
          tester
                  .widgetList<DecoratedBox>(
                    find.descendant(
                      of: item,
                      matching: find.byType(DecoratedBox),
                    ),
                  )
                  .firstWhere(
                    (d) =>
                        d.decoration is BoxDecoration &&
                        (d.decoration as BoxDecoration).boxShadow != null,
                  )
                  .decoration
              as BoxDecoration;

      // The press motion animates over the 100ms decree duration (the
      // selection-highlight sibling shares the same decree window).
      final pressContainer = tester
          .widgetList<AnimatedContainer>(animated)
          .firstWhere((c) => c.duration == const Duration(milliseconds: 100));
      expect(pressContainer.duration, const Duration(milliseconds: 100));

      // Rest: full cast shadow (4,4), zero blur, no translation.
      expect(
        shadowDecoration().boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(4, 4))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );
      expect(translationX(), 0);

      // Press: the block travels to (4,4) and the shadow is gone; halfway
      // through the decree window it sits between rest and pressed. (The first
      // pump applies the press; the tween then runs over the next 100ms.)
      final gesture = await tester.startGesture(
        tester.getRect(item).center,
        kind: PointerDeviceKind.mouse,
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      expect(translationX(), moreOrLessEquals(2, epsilon: 0.1));
      await tester.pump(const Duration(milliseconds: 60));
      expect(translationX(), 4);
      expect(shadowDecoration().boxShadow, isEmpty);
      await gesture.up();
      await tester.pump();
    },
  );

  testWidgets(
    'a hover shifts the dock item (2,2) and shrinks the cast to (2,2)',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await pumpDock(tester, focus: focus);

      final item = find.byType(PressableBlock).first;
      final transform = find.descendant(
        of: item,
        matching: find.byType(Transform),
      );
      double translationX() =>
          tester.widget<Transform>(transform).transform.getTranslation().x;
      BoxDecoration shadowDecoration() =>
          tester
                  .widgetList<DecoratedBox>(
                    find.descendant(
                      of: item,
                      matching: find.byType(DecoratedBox),
                    ),
                  )
                  .firstWhere(
                    (d) =>
                        d.decoration is BoxDecoration &&
                        (d.decoration as BoxDecoration).boxShadow != null,
                  )
                  .decoration
              as BoxDecoration;

      // Park a mouse pointer far from the dock, then glide it over the item:
      // translate (2,2) with the cast shrunk to castSm.
      final mouse = await tester.startGesture(
        const Offset(20, 300),
        kind: PointerDeviceKind.mouse,
      );
      await mouse.up();
      await mouse.moveTo(tester.getRect(item).center);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      expect(translationX(), moreOrLessEquals(1, epsilon: 0.1));
      await tester.pump(const Duration(milliseconds: 60));
      expect(translationX(), 2);
      expect(
        shadowDecoration().boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(2, 2))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );

      // Glide off: the block settles back on the full cast.
      await mouse.moveTo(const Offset(20, 300));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 120));
      expect(translationX(), 0);
      expect(shadowDecoration().boxShadow!.single.offset, const Offset(4, 4));
      await mouse.removePointer();
    },
  );

  testWidgets(
    'the dock is a Cobalt-Neo block: 0 radius, 2px ink border, cast shadow',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await pumpDock(tester, focus: focus);

      final dock = tester.widget<Container>(
        find.byWidgetPredicate(
          (w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration! as BoxDecoration).color == const Color(0xFF3B5BFF),
        ),
      );
      final decoration = dock.decoration! as BoxDecoration;
      expect(decoration.borderRadius, BorderRadius.zero);
      final border = decoration.border! as Border;
      expect(border.top.width, 2);
      expect(
        decoration.boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(4, 4))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );
    },
  );
}
