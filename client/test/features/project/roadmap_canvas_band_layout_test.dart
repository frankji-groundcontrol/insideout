import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// Design QA 2026-11-21: a lone leaf sat shifted right inside its band (the fold
// gutter was reserved with nothing to fold), and the band ran about a dock's
// height past its last card (the list inherited the page's MediaQuery bottom
// padding).

RoadmapNode _node(String id, String? parent) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: id,
  description: '',
  status: 'pending',
  position: 0,
);

Future<void> _pump(WidgetTester tester, List<RoadmapNode> nodes) async {
  final controller = ScrollController();
  addTearDown(controller.dispose);
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore()),
      child: MaterialApp(
        home: MediaQuery(
          // The page reserves the dock's height as bottom padding.
          data: const MediaQueryData(
            size: Size(800, 600),
            padding: EdgeInsets.only(bottom: 90),
          ),
          child: Scaffold(
            body: RoadmapCanvas(
              nodes: nodes,
              controller: controller,
              onToggle: (_) {},
              tileBuilder: (node, _) =>
                  SizedBox(key: Key('tile-${node.id}'), height: 100),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Rect _band(WidgetTester tester) => tester.getRect(
  find
      .descendant(
        of: find.byType(RoadmapCanvas),
        matching: find.byType(ListView),
      )
      .first,
);

void main() {
  testWidgets('a band of leaves reserves no fold gutter', (tester) async {
    await _pump(tester, [_node('A', null)]);
    final band = _band(tester);
    final tile = tester.getRect(find.byKey(const Key('tile-A')));
    expect(tile.left - band.left, lessThanOrEqualTo(12));
    expect(band.right - tile.right, lessThanOrEqualTo(12));
  });

  testWidgets('a band ends at its last card, not a dock height below it', (
    tester,
  ) async {
    await _pump(tester, [_node('A', null)]);
    final band = _band(tester);
    final tile = tester.getRect(find.byKey(const Key('tile-A')));
    expect(band.bottom - tile.bottom, lessThanOrEqualTo(12));
  });

  testWidgets('a band with a branch keeps the fold toggle beside it', (
    tester,
  ) async {
    await _pump(tester, [_node('A', null), _node('a1', 'A')]);
    expect(find.byKey(const Key('roadmap-collapse-A')), findsOneWidget);
    final toggle = tester.getRect(find.byKey(const Key('roadmap-collapse-A')));
    final tile = tester.getRect(find.byKey(const Key('tile-A')));
    expect(toggle.right, lessThanOrEqualTo(tile.left));
  });
}
