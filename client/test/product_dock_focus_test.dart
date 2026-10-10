import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/theme/block_focus.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/widgets/pressable_block.dart';

import '_dock_test_helpers.dart';

// The dock is a brand-filled block, and the block focus ring used to be brand
// too: on the dock's top and left sides (over the dock fill, clear of the
// item's own cast shadow) it was brand on brand, 1.00:1, and the current item
// was a 16% btnFg wash, 1.37:1 against its neighbours. These pin a visible
// ring and a visible, announced current item in both themes.

Cobalt _palette(bool dark) => dark ? Cobalt.dark : Cobalt.light;

Future<void> _pump(WidgetTester tester, bool dark) async {
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await pumpDock(
    tester,
    focus: warmFocus(),
    theme: cobaltTheme(
      _palette(dark),
      brightness: dark ? Brightness.dark : Brightness.light,
    ),
  );
}

Finder _rings() => find.byWidgetPredicate(
  (w) => w is CustomPaint && w.foregroundPainter is CobaltFocusRing,
);

/// The padded fill block inside one dock item, where the current-item outline
/// is painted (PressableBlock's own AnimatedContainer only carries the cast).
AnimatedContainer _shell(WidgetTester tester, DockItem item) => tester
    .widgetList<AnimatedContainer>(
      find.descendant(
        of: find.byType(PressableBlock).at(item.index),
        matching: find.byType(AnimatedContainer),
      ),
    )
    .firstWhere((c) => c.padding != null);

double _contrast(Color a, Color b) {
  final hi = math.max(a.computeLuminance(), b.computeLuminance());
  final lo = math.min(a.computeLuminance(), b.computeLuminance());
  return (hi + 0.05) / (lo + 0.05);
}

void main() {
  for (final dark in [true, false]) {
    final theme = dark ? 'dark' : 'light';

    testWidgets('a focused dock item rings in a colour that shows on the dock '
        '($theme)', (tester) async {
      await _pump(tester, dark);
      expect(_rings(), findsNothing);

      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pumpAndSettle();
      expect(_rings(), findsOneWidget);

      final ring =
          (tester.widget<CustomPaint>(_rings()).foregroundPainter!
                  as CobaltFocusRing)
              .color;
      expect(
        _contrast(ring, _palette(dark).brand),
        greaterThanOrEqualTo(3.0),
        reason: 'the dock is filled with brand; a brand ring disappears on it',
      );
    });

    testWidgets('the current dock item is outlined 2px in btnFg, the others '
        'are not ($theme)', (tester) async {
      await _pump(tester, dark);

      for (final item in DockItem.values) {
        final outline = _shell(tester, item).foregroundDecoration;
        if (item == DockItem.roadmap) {
          expect(outline, isA<BoxDecoration>(), reason: 'the current item');
          expect(
            (outline! as BoxDecoration).border,
            Border.all(
              color: _palette(dark).btnFg,
              width: Cobalt.borderContainer,
            ),
          );
        } else {
          expect(outline, isNull, reason: item.name);
        }
      }
    });
  }

  testWidgets('only the current dock item is announced as selected', (
    tester,
  ) async {
    final handle = tester.ensureSemantics();
    addTearDown(handle.dispose);
    await _pump(tester, true);

    // The default location is the roadmap route.
    for (final item in DockItem.values) {
      expect(
        tester.getSemantics(find.byType(PressableBlock).at(item.index)),
        isSemantics(isSelected: item == DockItem.roadmap),
        reason: item.name,
      );
    }
  });
}
