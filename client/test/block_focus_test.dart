import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/block_focus.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/widgets/pressable_block.dart';

Widget host(Widget child, {bool dark = true}) {
  final palette = dark ? Cobalt.dark : Cobalt.light;
  return MaterialApp(
    theme: cobaltTheme(
      palette,
      brightness: dark ? Brightness.dark : Brightness.light,
    ),
    home: Scaffold(body: Center(child: child)),
  );
}

/// The cobalt focus ring, found by its painter so the assertion cannot be
/// satisfied by an unrelated CustomPaint in the widget tree.
Finder rings() => find.byWidgetPredicate(
  (w) => w is CustomPaint && w.foregroundPainter is CobaltFocusRing,
);

Future<void> focusFirst(WidgetTester tester) async {
  await tester.sendKeyEvent(LogicalKeyboardKey.tab);
  await tester.pumpAndSettle();
}

/// One theme per call: a fresh `testWidgets` gets a fresh tree, so the ring
/// state cannot leak from the previous theme the way a loop would let it.
Future<void> expectCobaltRing(WidgetTester tester, bool dark) async {
  await tester.pumpWidget(
    host(
      BlockButton(label: 'Go', onPressed: () {}),
      dark: dark,
    ),
  );
  await tester.pumpAndSettle();
  expect(rings(), findsNothing, reason: 'unfocused, dark=$dark');

  await focusFirst(tester);
  expect(rings(), findsOneWidget, reason: 'focused, dark=$dark');

  final painter =
      tester.widgetList<CustomPaint>(rings()).first.foregroundPainter!
          as CobaltFocusRing;
  expect(
    painter.color,
    (dark ? Cobalt.dark : Cobalt.light).brand,
    reason: 'DESIGN.md: focus is a cobalt ring',
  );
}

void main() {
  testWidgets('Tab reaches a BlockButton and Enter fires onPressed', (
    tester,
  ) async {
    var taps = 0;
    await tester.pumpWidget(
      host(BlockButton(label: 'Go', onPressed: () => taps++)),
    );
    await tester.pumpAndSettle();
    expect(taps, 0);

    await focusFirst(tester);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(taps, 1);
  });

  testWidgets('Space fires onPressed on a focused BlockButton', (tester) async {
    var taps = 0;
    await tester.pumpWidget(
      host(BlockButton(label: 'Go', onPressed: () => taps++)),
    );
    await tester.pumpAndSettle();

    await focusFirst(tester);
    await tester.sendKeyEvent(LogicalKeyboardKey.space);
    await tester.pumpAndSettle();
    expect(taps, 1);
  });

  testWidgets('the focused block paints a cobalt ring in the dark theme', (
    tester,
  ) async {
    await expectCobaltRing(tester, true);
  });

  testWidgets('the focused block paints a cobalt ring in the light theme', (
    tester,
  ) async {
    await expectCobaltRing(tester, false);
  });

  testWidgets('the ring does not change the measured block size', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(
        BlockButton(label: 'Go', size: BlockButtonSize.md, onPressed: () {}),
      ),
    );
    await tester.pumpAndSettle();
    final rest = tester.getSize(find.byType(BlockButton));
    expect(rest.height, BlockButtonSize.md.height);

    await focusFirst(tester);
    expect(rings(), findsOneWidget);
    expect(tester.getSize(find.byType(BlockButton)), rest);
  });

  testWidgets('a disabled BlockButton stays out of focus traversal', (
    tester,
  ) async {
    await tester.pumpWidget(host(const BlockButton(label: 'No')));
    await tester.pumpAndSettle();

    // A disabled block is not wrapped in BlockFocus at all, so there is no
    // focus node to reach — that is what keeps it out of Tab traversal.
    expect(
      find.descendant(
        of: find.byType(BlockButton),
        matching: find.byType(BlockFocus),
      ),
      findsNothing,
    );

    await focusFirst(tester);
    expect(rings(), findsNothing);
  });

  testWidgets('BlockIconButton and PressableBlock activate from the keyboard', (
    tester,
  ) async {
    var icons = 0;
    var blocks = 0;
    await tester.pumpWidget(
      host(
        Column(
          children: [
            BlockIconButton(icon: Icons.translate, onPressed: () => icons++),
            PressableBlock(
              onTap: () => blocks++,
              child: const SizedBox(width: 80, height: 40),
            ),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();

    await focusFirst(tester);
    expect(rings(), findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(icons, 1);

    await focusFirst(tester);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(blocks, 1);
  });
}
