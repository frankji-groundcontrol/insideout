import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';

// Block-craft follow-up: a default PopupMenuButton (no per-site styling) must
// open the block menu — square, 2px bordered, zero elevation, no ripple —
// because the treatment lives in the theme, not at the call sites. Rendered,
// not asserted on ThemeData, so a theme that is defined but never consumed
// still fails.
Future<void> _openMenu(WidgetTester tester, Cobalt palette, bool dark) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: cobaltTheme(
        palette,
        brightness: dark ? Brightness.dark : Brightness.light,
      ),
      home: Scaffold(
        body: Center(
          child: PopupMenuButton<String>(
            tooltip: 'More',
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'a', child: Text('Alpha')),
              PopupMenuItem(value: 'b', child: Text('Beta')),
            ],
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.byTooltip('More'));
  await tester.pumpAndSettle();
}

void main() {
  for (final dark in [false, true]) {
    final label = dark ? 'dark' : 'light';
    final palette = dark ? Cobalt.dark : Cobalt.light;

    testWidgets('a default PopupMenuButton opens a block menu ($label)', (
      tester,
    ) async {
      await _openMenu(tester, palette, dark);

      // The nearest Material above a menu item is the menu card itself.
      final menu = tester.widget<Material>(
        find
            .ancestor(of: find.text('Alpha'), matching: find.byType(Material))
            .first,
      );
      expect(menu.type, MaterialType.card);
      final shape = menu.shape! as RoundedRectangleBorder;
      expect(shape.borderRadius, BorderRadius.zero);
      expect(shape.side.width, Cobalt.borderContainer);
      expect(shape.side.color, palette.strokeSubtle);
      expect(menu.elevation, 0);
      expect(menu.color, palette.surfaceRaised);
      expect(menu.surfaceTintColor, Colors.transparent);

      // Item ink is switched off for the whole app: the block grammar presses,
      // it does not ripple. PopupMenuThemeData cannot carry a splash, so this
      // rides on ThemeData and reaches the menu through the captured theme.
      final itemTheme = Theme.of(tester.element(find.text('Alpha')));
      expect(itemTheme.splashFactory, same(NoSplash.splashFactory));
      expect(itemTheme.splashColor, Colors.transparent);
      expect(itemTheme.highlightColor, Colors.transparent);
    });

    testWidgets('the default trigger highlights as a square, not a circle '
        '($label)', (tester) async {
      await _openMenu(tester, palette, dark);

      // PopupMenuButton without a `child` is a stock IconButton, whose M3
      // hover/press highlight follows a stadium shape.
      final trigger = tester.widget<InkWell>(
        find.descendant(
          of: find.byType(IconButton),
          matching: find.byType(InkWell),
        ),
      );
      expect(
        trigger.customBorder,
        isA<RoundedRectangleBorder>().having(
          (b) => b.borderRadius,
          'borderRadius',
          BorderRadius.zero,
        ),
      );
    });
  }
}
