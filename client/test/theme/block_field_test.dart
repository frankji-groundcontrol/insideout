import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/block_field.dart';
import 'package:insideout/theme/cobalt.dart';

import '_block_test_helpers.dart';

void main() {
  testWidgets(
    'BlockField: 44 tall, label above, focus flips the border to brand',
    (tester) async {
      await tester.pumpWidget(
        host(
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
      host(
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
      host(
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
}

Finder _fieldBox() => find
    .ancestor(of: find.byType(TextField), matching: find.byType(Container))
    .first;

Size _fieldBoxSize(WidgetTester tester) => tester.getSize(_fieldBox());

Color _fieldBorder(WidgetTester tester) {
  final deco =
      tester.widget<Container>(_fieldBox()).decoration! as BoxDecoration;
  return (deco.border! as Border).top.color;
}
