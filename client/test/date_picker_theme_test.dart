import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';

// Block-craft follow-up: showDatePicker (the roadmap deadline) rendered stock
// M3 round chrome — a 28px dialog, circular day cells, a stadium year cell.
// The picker is themed once, at the theme level; the call site is untouched,
// so this drives a bare showDatePicker under cobaltTheme and inspects what
// actually paints.
const _selected = 15;
const _today = 10;

Future<void> _openPicker(WidgetTester tester, bool dark) async {
  await tester.binding.setSurfaceSize(const Size(420, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MaterialApp(
      theme: cobaltTheme(
        dark ? Cobalt.dark : Cobalt.light,
        brightness: dark ? Brightness.dark : Brightness.light,
      ),
      home: Builder(
        builder: (context) => Scaffold(
          body: Center(
            child: TextButton(
              onPressed: () => showDatePicker(
                context: context,
                initialDate: DateTime(2026, 10, _selected),
                currentDate: DateTime(2026, 10, _today),
                firstDate: DateTime(2026),
                lastDate: DateTime(2027),
              ),
              child: const Text('open'),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text('open'));
  await tester.pumpAndSettle();
}

/// The decoration painted behind a day number in the month grid.
ShapeDecoration _dayDecoration(WidgetTester tester, int day) {
  final cell = find
      .ancestor(of: find.text('$day'), matching: find.byType(Ink))
      .first;
  return tester.widget<Ink>(cell).decoration! as ShapeDecoration;
}

/// The decoration painted behind a year in the year grid.
ShapeDecoration _yearDecoration(WidgetTester tester, int year) {
  final cell = find
      .ancestor(of: find.text('$year'), matching: find.byType(Container))
      .first;
  return tester.widget<Container>(cell).decoration! as ShapeDecoration;
}

Matcher _square([double? borderWidth]) => isA<RoundedRectangleBorder>()
    .having((s) => s.borderRadius, 'borderRadius', BorderRadius.zero)
    .having((s) => s.side.width, 'side.width', borderWidth ?? anything);

void main() {
  for (final dark in [false, true]) {
    final label = dark ? 'dark' : 'light';
    final palette = dark ? Cobalt.dark : Cobalt.light;

    testWidgets('the picker dialog is a flat, square, 2px block ($label)', (
      tester,
    ) async {
      await _openPicker(tester, dark);

      final dialog = tester.widget<Dialog>(find.byType(Dialog));
      expect(dialog.shape, _square(Cobalt.borderContainer));
      expect(
        (dialog.shape! as RoundedRectangleBorder).side.color,
        palette.strokeSubtle,
      );
      expect(dialog.elevation, 0);
      expect(dialog.backgroundColor, palette.surfaceRaised);
    });

    testWidgets('day cells are squares, not circles ($label)', (tester) async {
      await _openPicker(tester, dark);

      // A plain day, the selected day, and today (which also carries a 2px
      // outline). Each shape must be a zero-radius rectangle.
      expect(_dayDecoration(tester, 20).shape, _square());
      expect(_dayDecoration(tester, _selected).shape, _square());
      expect(
        _dayDecoration(tester, _today).shape,
        _square(Cobalt.borderContainer),
      );

      // The month arrows are stock IconButtons: their highlight is squared by
      // the same theme that squares the popup-menu trigger.
      final arrow = tester.widget<InkWell>(
        find.descendant(
          of: find.ancestor(
            of: find.byIcon(Icons.chevron_right),
            matching: find.byType(IconButton),
          ),
          matching: find.byType(InkWell),
        ),
      );
      expect(arrow.customBorder, _square());
    });

    testWidgets('year cells are squares, not stadiums ($label)', (
      tester,
    ) async {
      await _openPicker(tester, dark);
      await tester.tap(find.byIcon(Icons.arrow_drop_down));
      await tester.pumpAndSettle();

      // 2026 is the selected + current year; 2027 is a plain one.
      expect(_yearDecoration(tester, 2027).shape, _square());
      expect(_yearDecoration(tester, 2026).shape, _square());
    });
  }
}
