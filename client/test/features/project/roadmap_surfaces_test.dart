import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_band_scaffold.dart';
import 'package:insideout/features/project/roadmap_bands.dart';
import 'package:insideout/features/project/roadmap_progress_view.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_chip.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

// DESIGN.md reserves cobalt-950 for status-chip backgrounds, so no band slab
// may paint it; and the Progress view's item cards use the same card grammar
// (2px border, cast) as the Bands view's. Roadmap critique 2026-11-21: every
// slab is the neutral lane and Now alone is emphasised.

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

void _expectNeutralSlabs(WidgetTester tester, Cobalt palette) {
  final slabs = find.byKey(const Key('roadmap-band-slab'));
  expect(slabs, findsWidgets);
  for (final slab in tester.widgetList<Container>(slabs)) {
    final color = (slab.decoration! as BoxDecoration).color;
    expect(color, isNot(Cobalt.hue950));
    expect(color, palette.surfaceSunken);
  }
  final bands = tester.widgetList<RoadmapBandScaffold>(
    find.byType(RoadmapBandScaffold),
  );
  expect(
    [
      for (final b in bands)
        if (b.now) b.label,
    ],
    ['Now'],
    reason: 'Now is the only emphasised band',
  );
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

    _expectNeutralSlabs(tester, Cobalt.dark);
  });

  testWidgets('Progress view: no band slab is cobalt-950 in dark mode', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapProgressView(progress: _progress, emptyLabel: 'empty'),
      Brightness.dark,
    );

    _expectNeutralSlabs(tester, Cobalt.dark);
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

  // Pressure is a shade chip: danger for overdue and high risk, neutral for
  // near, and "normal" is not printed (the Bands cards already hide it).
  testWidgets('Progress view: pressure is a chip and normal is hidden', (
    tester,
  ) async {
    ProgressItem dated(String id, String pressure) => ProgressItem(
      id: id,
      title: id,
      status: 'in_progress',
      deadline: '2026-12-01T00:00:00Z',
      pressure: pressure,
    );
    await _pump(
      tester,
      RoadmapProgressView(
        progress: RoadmapProgress(
          now: [
            dated('a', 'near'),
            dated('b', 'overdue'),
            dated('c', 'high_risk'),
            dated('d', 'normal'),
          ],
          needsDeadline: [],
          next: [],
          doneCount: 0,
        ),
        emptyLabel: 'empty',
      ),
      Brightness.dark,
    );

    ShadeTone tone(String label) => tester
        .widget<ShadeChip>(
          find.ancestor(of: find.text(label), matching: find.byType(ShadeChip)),
        )
        .tone;
    expect(tone('near'), ShadeTone.neutral);
    expect(tone('overdue'), ShadeTone.danger);
    expect(tone('high risk'), ShadeTone.danger);
    expect(find.text('normal'), findsNothing);
  });

  testWidgets('Progress view hides an empty Needs deadline band', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapProgressView(
        progress: RoadmapProgress(
          now: [_item('a', 'Wire the flow')],
          needsDeadline: [],
          next: [],
          doneCount: 0,
        ),
        emptyLabel: 'empty',
      ),
      Brightness.light,
    );

    expect(find.text('NEEDS DEADLINE'), findsNothing);
    expect(find.text('NOW'), findsOneWidget);
    expect(find.text('NEXT'), findsOneWidget);
  });
}
