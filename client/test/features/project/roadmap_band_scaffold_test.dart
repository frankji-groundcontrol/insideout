import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_band_scaffold.dart';

// Design QA 2026-11-21: cards in one band row ended at different heights, so
// the row read as ragged. A row now takes its tallest card's height.

Widget _card(String key, double contentHeight) => SizedBox(
  width: 240,
  child: Container(
    key: Key(key),
    decoration: BoxDecoration(border: Border.all()),
    child: Column(children: [SizedBox(height: contentHeight)]),
  ),
);

Future<void> _pump(WidgetTester tester, double width) async {
  tester.view.physicalSize = const Size(1200, 800);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: SizedBox(
            width: width,
            child: RoadmapBandScaffold(
              label: 'Now',
              color: Colors.black,
              itemCards: [_card('a', 80), _card('b', 140), _card('c', 60)],
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('cards in one row share the tallest card height', (tester) async {
    await _pump(tester, 1000);
    final a = tester.getRect(find.byKey(const Key('a')));
    final b = tester.getRect(find.byKey(const Key('b')));
    final c = tester.getRect(find.byKey(const Key('c')));
    expect(a.top, b.top);
    expect(a.height, b.height);
    expect(c.height, b.height);
  });

  testWidgets('cards wrap to a new row when the band is narrow', (
    tester,
  ) async {
    // 32 padding + 72 label leaves room for two 240 cards and a 12 gap.
    await _pump(tester, 32 + 72 + 240 * 2 + 12);
    final a = tester.getRect(find.byKey(const Key('a')));
    final b = tester.getRect(find.byKey(const Key('b')));
    final c = tester.getRect(find.byKey(const Key('c')));
    expect(a.top, b.top);
    expect(c.top, b.bottom + 12);
    expect(c.height, lessThan(b.height));
  });
}
