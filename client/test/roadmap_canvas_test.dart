import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [
    RoadmapNode(
      id: 'r1',
      projectId: 'p1',
      parentId: null,
      title: 'Band A',
      description: '',
      status: 'pending',
      position: 0,
    ),
    RoadmapNode(
      id: 'r2',
      projectId: 'p1',
      parentId: null,
      title: 'Band B',
      description: '',
      status: 'pending',
      position: 1,
    ),
    RoadmapNode(
      id: 'c1',
      projectId: 'p1',
      parentId: 'r1',
      title: 'A leaf',
      description: '',
      status: 'in_progress',
      position: 0,
    ),
  ];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress.fromJson({
        'now': [
          {
            'id': 'c1',
            'title': 'A leaf',
            'status': 'in_progress',
            'deadline': '2026-10-10T00:00:00Z',
            'pressure': 'near',
          },
        ],
        'needsDeadline': [],
        'next': [],
        'doneCount': 0,
      });

  final _parked = Completer<void>();

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress,
  ) {
    return _parked.future;
  }
}

void main() {
  testWidgets('roadmap page toggles to the canvas with bands and minimap', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
      ),
    );
    await tester.pumpAndSettle();

    // Progress view is the default (server-truth buckets): the deadlined
    // in-progress leaf shows there, no canvas yet.
    expect(find.text('A leaf'), findsOneWidget);
    expect(find.byType(RoadmapMinimap), findsNothing);

    // Toggle to bands.
    await tester.tap(find.text('Bands'));
    await tester.pumpAndSettle();
    expect(find.text('Band A'), findsWidgets);

    // Toggle to canvas.
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();

    expect(find.byType(RoadmapMinimap), findsOneWidget);
    expect(find.byType(RoadmapCanvas), findsOneWidget);
    // Both sibling bands render, with the child inside its band.
    expect(find.text('Band A'), findsWidgets);
    expect(find.text('Band B'), findsWidgets);
    expect(find.text('A leaf'), findsOneWidget);

    // Minimap tap jumps to a band without error.
    await tester.tap(find.byType(RoadmapMinimap));
    await tester.pumpAndSettle();
    expect(find.byType(RoadmapCanvas), findsOneWidget);
  });

  testWidgets(
    'minimap marks encode state by shape and semantics, not color alone',
    (tester) async {
      final nodes = [
        RoadmapNode(
          id: 'r1',
          projectId: 'p1',
          parentId: null,
          title: 'Alpha',
          description: '',
          status: 'done',
          position: 0,
        ),
        RoadmapNode(
          id: 'c1',
          projectId: 'p1',
          parentId: 'r1',
          title: 'todo',
          description: '',
          status: 'pending',
          position: 0,
        ),
        RoadmapNode(
          id: 'c2',
          projectId: 'p1',
          parentId: 'r1',
          title: 'doing',
          description: '',
          status: 'in_progress',
          position: 1,
        ),
        RoadmapNode(
          id: 'c3',
          projectId: 'p1',
          parentId: 'r1',
          title: 'blocked',
          description: '',
          status: 'locked',
          position: 2,
        ),
        RoadmapNode(
          id: 'r2',
          projectId: 'p1',
          parentId: null,
          title: 'Beta',
          description: '',
          status: 'locked',
          position: 1,
        ),
      ];

      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: RoadmapMinimap(nodes: nodes, onTapBand: (_) {}),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Screen readers (and hover tooltips) announce each mark's state.
      expect(find.bySemanticsLabel('done'), findsOneWidget);
      expect(find.bySemanticsLabel('pending'), findsOneWidget);
      expect(find.bySemanticsLabel('in progress'), findsOneWidget);
      expect(find.bySemanticsLabel('locked'), findsNWidgets(2));

      // Fingerprints omit color. Two statuses that share one have collapsed
      // to the same paint, even if their colors still differ.
      final done = _paintFingerprint(tester, 'done');
      final pending = _paintFingerprint(tester, 'pending');
      final progress = _paintFingerprint(tester, 'in progress');
      final locked = _paintFingerprint(tester, 'locked');
      expect({done, pending, progress, locked}, hasLength(4));
      expect(done, 'square-filled');
      expect(pending, 'square-hollow');
      expect(progress, 'square-half:0.5');
      expect(locked, 'square-dashed');

      final dashes = tester
          .widgetList<CustomPaint>(
            find.byWidgetPredicate(
              (w) => w is CustomPaint && w.painter is MinimapDashMarkPainter,
            ),
          )
          .toList();
      expect(dashes, hasLength(2));
      final dash = dashes.first.painter! as MinimapDashMarkPainter;
      expect(dash.color, Cobalt.light.strokeStrong);
      expect(
        dash.shouldRepaint(MinimapDashMarkPainter(Cobalt.light.strokeStrong)),
        isFalse,
      );
      expect(
        dash.shouldRepaint(MinimapDashMarkPainter(Cobalt.dark.strokeStrong)),
        isTrue,
      );
      semantics.dispose();
    },
  );
}

/// Color-free paint signature for every mark that announces [label].
/// All marks with that label must share one signature.
String _paintFingerprint(WidgetTester tester, String label) {
  final marks = find.descendant(
    of: find.bySemanticsLabel(label),
    matching: find.byWidgetPredicate(_isStatusMark),
  );
  expect(marks, findsWidgets);
  final fingerprints = tester.widgetList(marks).map(_fingerprintOf).toSet();
  expect(fingerprints, hasLength(1));
  return fingerprints.single;
}

bool _isStatusMark(Widget w) {
  if (w is CustomPaint && w.painter is MinimapDashMarkPainter) return true;
  return w is Container && w.constraints?.maxHeight == 4;
}

String _fingerprintOf(Widget w) {
  if (w is CustomPaint && w.painter is MinimapDashMarkPainter) {
    final sized = w.child;
    expect(sized, isA<SizedBox>());
    final box = sized! as SizedBox;
    expect(box.width, double.infinity);
    expect(box.height, 4);
    return 'square-dashed';
  }
  final mark = w as Container;
  final decoration = mark.decoration! as BoxDecoration;
  expect(decoration.shape, BoxShape.rectangle);
  expect(decoration.borderRadius, BorderRadius.zero);
  if (decoration.color != null &&
      decoration.border == null &&
      mark.child == null) {
    return 'square-filled';
  }
  if (decoration.color == null &&
      decoration.border != null &&
      mark.child is FractionallySizedBox) {
    final half = mark.child! as FractionallySizedBox;
    expect(half.alignment, Alignment.centerLeft);
    return 'square-half:${half.widthFactor}';
  }
  if (decoration.color == null &&
      decoration.border != null &&
      mark.child == null) {
    return 'square-hollow';
  }
  return 'collapsed:${decoration.color != null}:${decoration.border != null}:${mark.child?.runtimeType}';
}
