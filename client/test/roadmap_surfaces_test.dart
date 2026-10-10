import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_band_scaffold.dart';
import 'package:insideout/features/project/roadmap_bands.dart';
import 'package:insideout/features/project/roadmap_progress_view.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

// DESIGN.md reserves cobalt-950 for status-chip backgrounds, so no band slab
// may paint it; and the Progress view's item cards use the same card grammar
// (2px border, cast) as the Bands view's.

ProgressItem _item(String id, String title) =>
    ProgressItem(id: id, title: title, status: 'in_progress');

final _progress = RoadmapProgress(
  now: [_item('a', 'Wire the flow')],
  needsDeadline: [_item('b', 'Pick a date')],
  next: [_item('c', 'Sketch copy')],
  doneCount: 1,
  doneItems: [_item('d', 'Ship the importer')],
);

RoadmapNode _node(String id, String status, int position) => RoadmapNode(
  id: id,
  projectId: 'p1',
  title: id,
  description: '',
  status: status,
  position: position,
);

Future<void> _pump(WidgetTester tester, Widget body, Brightness mode) async {
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
      child: MaterialApp(
        theme: ThemeData(brightness: mode),
        home: Scaffold(body: body),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void _expectNoCobalt950Slab(WidgetTester tester) {
  final bands = tester.widgetList<RoadmapBandScaffold>(
    find.byType(RoadmapBandScaffold),
  );
  expect(bands, isNotEmpty);
  for (final band in bands) {
    expect(
      band.color,
      isNot(Cobalt.hue950),
      reason: 'the ${band.label} band paints cobalt-950',
    );
  }
}

void main() {
  testWidgets('Bands view: no band slab is cobalt-950 in dark mode', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapBands(
        projectId: 'p1',
        onReload: () async {},
        nodes: [
          _node('now', 'in_progress', 0),
          _node('next', 'pending', 1),
          _node('later', 'locked', 2),
        ],
      ),
      Brightness.dark,
    );

    _expectNoCobalt950Slab(tester);
  });

  testWidgets('Progress view: no band slab is cobalt-950 in dark mode', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapProgressView(progress: _progress, emptyLabel: 'empty'),
      Brightness.dark,
    );

    _expectNoCobalt950Slab(tester);
  });

  testWidgets('Progress view items are field cards, not bare containers', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapProgressView(progress: _progress, emptyLabel: 'empty'),
      Brightness.light,
    );

    for (final title in [
      'Wire the flow',
      'Pick a date',
      'Sketch copy',
      'Ship the importer',
    ]) {
      expect(
        find.ancestor(of: find.text(title), matching: find.byType(FieldCard)),
        findsOneWidget,
        reason: '$title is not on a FieldCard',
      );
    }
  });

  // The 'normal'/'near' pressure label is 12px bold — normal text, so it owes
  // 4.5:1 on the raised card. hue500 measures 3.53:1 in dark; the dark
  // brand-text token (fgBrand, cobalt-200) is the one that clears the bar.
  testWidgets('Progress view: pressure label clears 4.5:1 on the dark card', (
    tester,
  ) async {
    final near = ProgressItem(
      id: 'a',
      title: 'Wire the flow',
      status: 'in_progress',
      pressure: 'near',
    );
    await _pump(
      tester,
      RoadmapProgressView(
        progress: RoadmapProgress(
          now: [near],
          needsDeadline: [],
          next: [],
          doneCount: 0,
        ),
        emptyLabel: 'empty',
      ),
      Brightness.dark,
    );

    final label = tester.widget<Text>(find.text('near'));
    final ink = label.style!.color!;
    expect(
      _contrast(ink, Cobalt.dark.surfaceRaised),
      greaterThanOrEqualTo(4.5),
      reason:
          'pressure ink ${ink.toARGB32()} on raised '
          '${Cobalt.dark.surfaceRaised.toARGB32()}',
    );
  });
}

double _channel(double v) =>
    v <= 0.03928 ? v / 12.92 : math.pow((v + 0.055) / 1.055, 2.4).toDouble();

double _luminance(Color c) =>
    0.2126 * _channel(c.r) + 0.7152 * _channel(c.g) + 0.0722 * _channel(c.b);

double _contrast(Color a, Color b) {
  final la = _luminance(a);
  final lb = _luminance(b);
  final hi = math.max(la, lb);
  final lo = math.min(la, lb);
  return (hi + 0.05) / (lo + 0.05);
}
