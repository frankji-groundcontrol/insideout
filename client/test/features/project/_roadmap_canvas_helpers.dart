import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/features/project/roadmap_step_flow.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

/// A pending node titled by its id.
RoadmapNode canvasNode(
  String id,
  String? parent, {
  int position = 0,
  List<String> dependsOn = const [],
}) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: id,
  description: '',
  status: 'pending',
  position: position,
  dependsOn: dependsOn,
);

/// A chain of [n] steps under one root R: each waits on the one before.
List<RoadmapNode> canvasChain(int n) => [
  canvasNode('R', null),
  for (var i = 0; i < n; i++)
    canvasNode('S$i', 'R', position: i, dependsOn: [if (i > 0) 'S${i - 1}']),
];

/// Pumps the Canvas over [all], its steps read as the page reads them.
Future<void> pumpCanvas(
  WidgetTester tester,
  List<RoadmapNode> all, {
  List<RoadmapNode>? steps,
  Size size = const Size(1400, 900),
  List<CursorDot> cursors = const [],
  Map<String, String> titles = const {},
}) async {
  await tester.binding.setSurfaceSize(size);
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
      child: MaterialApp(
        home: Scaffold(
          body: RoadmapCanvas(
            projectId: 'p1',
            steps: steps ?? roadmapSteps(all),
            all: all,
            titles: {for (final n in all) n.id: n.title, ...titles},
            cursors: cursors,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Finder stepBox(String id) => find.byKey(Key('roadmap-step-$id'));

/// The squared block of step [id].
Rect stepCard(WidgetTester tester, String id) => tester.getRect(
  find.descendant(of: stepBox(id), matching: find.byType(FieldCard)),
);

/// The canvas view's transformation.
Matrix4 canvasMatrix(WidgetTester tester) => tester
    .widget<InteractiveViewer>(find.byType(InteractiveViewer))
    .transformationController!
    .value;

/// The flow arrows the Canvas paints.
List<({String from, String to, int count})> canvasEdges(WidgetTester tester) =>
    (tester
                .widget<CustomPaint>(
                  find.byWidgetPredicate(
                    (w) => w is CustomPaint && w.painter is RoadmapFlowPainter,
                  ),
                )
                .painter!
            as RoadmapFlowPainter)
        .edges;
