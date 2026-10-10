import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_chip.dart';

// The ShadeChip's padding is not ad hoc: both densities sit on the named 4px
// scale, so chips densify or breathe with the system instead of per widget.

void main() {
  for (final (compact, horizontal, vertical) in [
    (true, Cobalt.space2, Cobalt.space1),
    (false, Cobalt.space3, Cobalt.space1),
  ]) {
    testWidgets('ShadeChip pads on the 4px scale (compact: $compact)', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: ShadeChip(
                label: 'x',
                tone: ShadeTone.pending,
                compact: compact,
              ),
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
      expect(
        chip.padding,
        EdgeInsets.symmetric(horizontal: horizontal, vertical: vertical),
        reason: 'compact: $compact pads off the 4px scale',
      );
    });
  }
}
