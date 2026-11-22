import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_band_scaffold.dart';
import 'package:insideout/features/project/roadmap_bands.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

// Roadmap critique 2026-11-21: Bands' Later held every unplaced node, done
// and validated work mixed in. Finished work now has its own Done band and
// Later is grouped under each top-level branch (a lone root's children).

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );
}

RoadmapNode _node(
  String id,
  String title,
  String? parent, {
  String status = 'pending',
  int position = 0,
}) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: title,
  description: '',
  status: status,
  position: position,
);

final _nodes = [
  _node('r', 'Ship', null),
  _node('a', 'Payments', 'r'),
  _node('a1', 'Refunds', 'a'),
  _node('a2', 'Checked receipts', 'a', status: 'validated', position: 1),
  _node('b', 'Search', 'r', position: 1),
  _node('b1', 'Old import', 'b', status: 'done'),
  _node('c', 'Onboarding', 'r', position: 2),
  _node('c1', 'Welcome mail', 'c'),
];

Future<void> _pump(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
  await tester.binding.setSurfaceSize(const Size(1400, 2000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: MaterialApp(
        theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
        home: Scaffold(
          body: RoadmapBands(
            projectId: 'p1',
            nodes: _nodes,
            onReload: () async {},
            progress: RoadmapProgress(
              now: const [],
              needsDeadline: const [],
              next: [ProgressItem(id: 'b', title: 'Search', status: 'pending')],
              doneCount: 2,
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Finder _heading(String title) =>
    find.byKey(ValueKey('roadmap-band-group-$title'));

void main() {
  testWidgets('done and validated work sits in a Done band after Later', (
    tester,
  ) async {
    await _pump(tester);
    double dy(Finder f) => tester.getTopLeft(f).dy;

    expect(find.text('DONE'), findsOneWidget);
    expect(dy(find.text('DONE')), greaterThan(dy(find.text('Welcome mail'))));
    expect(dy(find.text('Old import')), greaterThan(dy(find.text('DONE')) - 1));
    expect(
      dy(find.text('Checked receipts')),
      greaterThan(dy(find.text('DONE')) - 1),
    );
  });

  testWidgets('Later groups its cards under each top-level branch', (
    tester,
  ) async {
    await _pump(tester);
    double dy(Finder f) => tester.getTopLeft(f).dy;

    // The lone root leads ungrouped; then one heading per branch with work.
    expect(dy(find.text('Ship')), lessThan(dy(_heading('Payments'))));
    expect(dy(_heading('Payments')), lessThan(dy(find.text('Refunds'))));
    expect(dy(find.text('Refunds')), lessThan(dy(_heading('Onboarding'))));
    expect(dy(_heading('Onboarding')), lessThan(dy(find.text('Welcome mail'))));
    // Search's only unfinished node is in Next, so Later gives it no group.
    expect(_heading('Search'), findsNothing);
    expect(find.text('Search'), findsOneWidget);
  });

  testWidgets('only Now is emphasised; every slab is neutral', (tester) async {
    await _pump(tester);
    final bands = tester.widgetList<RoadmapBandScaffold>(
      find.byType(RoadmapBandScaffold),
    );
    expect([for (final b in bands) b.label], ['Now', 'Next', 'Later', 'Done']);
    expect(
      [
        for (final b in bands)
          if (b.now) b.label,
      ],
      ['Now'],
    );
  });
}
