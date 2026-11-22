import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_band_scaffold.dart';
import 'package:insideout/theme/cobalt.dart';

// Design QA 2026-11-21: cards in one band row ended at different heights, so
// the row read as ragged. A row now takes its tallest card's height.
// Roadmap critique 2026-11-21: cards fill the row up to the 1120 content
// cap, only Now is emphasised, and on a phone the label sits above the cards.

Widget _card(String key, double contentHeight) => Container(
  key: Key(key),
  decoration: BoxDecoration(border: Border.all()),
  child: Column(children: [SizedBox(height: contentHeight)]),
);

Future<void> _pump(
  WidgetTester tester,
  double width, {
  Size screen = const Size(1200, 800),
  bool now = false,
  List<RoadmapBandGroup>? groups,
}) async {
  tester.view.physicalSize = screen;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: SizedBox(
            width: width,
            child: groups != null
                ? RoadmapBandScaffold.grouped(label: 'Later', groups: groups)
                : RoadmapBandScaffold(
                    label: 'Now',
                    now: now,
                    itemCards: [
                      _card('a', 80),
                      _card('b', 140),
                      _card('c', 60),
                    ],
                  ),
          ),
        ),
      ),
    ),
  );
}

Rect _rect(WidgetTester tester, String key) =>
    tester.getRect(find.byKey(Key(key)));

void main() {
  testWidgets('cards in one row share the tallest card height', (tester) async {
    await _pump(tester, 1000);
    final a = _rect(tester, 'a');
    final b = _rect(tester, 'b');
    final c = _rect(tester, 'c');
    expect(a.top, b.top);
    expect(a.height, b.height);
    expect(c.height, b.height);
  });

  testWidgets('cards wrap to a new row when the band is narrow', (
    tester,
  ) async {
    // 32 padding + 72 label leaves room for two 220-plus cards and a gap.
    await _pump(tester, 32 + 72 + 240 * 2 + 12);
    final a = _rect(tester, 'a');
    final b = _rect(tester, 'b');
    final c = _rect(tester, 'c');
    expect(a.top, b.top);
    expect(c.top, b.bottom + 12);
    expect(c.height, lessThan(b.height));
  });

  testWidgets('cards fill the row: width is (row - gaps) / perRow', (
    tester,
  ) async {
    // Row 1000 - 32 - 72 = 896: three 220-plus cards fit, each (896 - 24) / 3.
    await _pump(tester, 1000);
    expect(_rect(tester, 'a').width, closeTo((896 - 2 * 12) / 3, 0.01));
    expect(_rect(tester, 'c').right, closeTo(1000 - 16, 0.01));
    expect(_rect(tester, 'b').left, _rect(tester, 'a').right + 12);
  });

  testWidgets('a wide band caps its content at the 1120 column', (
    tester,
  ) async {
    await _pump(tester, 1600, screen: const Size(1600, 900));
    final slab = tester.getRect(find.byKey(const Key('roadmap-band-slab')));
    expect(slab.width, Cobalt.contentMaxWidth);
    expect(slab.center.dx, closeTo(800, 0.5));
  });

  testWidgets('only a Now band carries the cobalt strip and label block', (
    tester,
  ) async {
    await _pump(tester, 1000, now: true);
    final strip = find.byKey(const Key('roadmap-band-now-strip'));
    expect(strip, findsOneWidget);
    expect(tester.getRect(strip).width, 4);
    final label = tester.widget<Container>(
      find.byKey(const Key('roadmap-band-label')),
    );
    expect((label.decoration! as BoxDecoration).color, Cobalt.light.brand);

    await _pump(tester, 1000);
    expect(find.byKey(const Key('roadmap-band-now-strip')), findsNothing);
    final plain = tester.widget<Container>(
      find.byKey(const Key('roadmap-band-label')),
    );
    expect(plain.decoration, isNull);
  });

  testWidgets('every slab paints the neutral lane', (tester) async {
    for (final now in [true, false]) {
      await _pump(tester, 1000, now: now);
      final slab = tester.widget<Container>(
        find.byKey(const Key('roadmap-band-slab')),
      );
      expect(
        (slab.decoration! as BoxDecoration).color,
        Cobalt.light.surfaceSunken,
      );
    }
  });

  testWidgets('on a phone the label sits above the cards', (tester) async {
    await _pump(tester, 390, screen: const Size(390, 844));
    final label = tester.getRect(find.text('NOW'));
    final a = _rect(tester, 'a');
    expect(a.top, greaterThan(label.bottom));
    expect(a.left, closeTo(label.left, 0.5));
    // The full row, not a row beside a 72px column.
    expect(a.width, closeTo(390 - 32, 0.5));
  });

  testWidgets('a grouped band heads each group', (tester) async {
    await _pump(
      tester,
      1000,
      groups: [
        RoadmapBandGroup(null, [_card('a', 40)]),
        RoadmapBandGroup('Payments', [_card('b', 40), _card('c', 40)]),
      ],
    );
    final heading = tester.getRect(find.text('Payments'));
    expect(heading.top, greaterThan(_rect(tester, 'a').bottom));
    expect(_rect(tester, 'b').top, greaterThan(heading.bottom));
    expect(_rect(tester, 'b').top, _rect(tester, 'c').top);
  });
}
