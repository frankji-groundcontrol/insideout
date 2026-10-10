import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// The minimap's mark labels are user-visible (screen readers, hover) and its
// column titles sit on the 12px meta floor, in every locale.

RoadmapNode _node(String id, String? parent, String status, int position) =>
    RoadmapNode(
      id: id,
      projectId: 'p1',
      parentId: parent,
      title: id,
      description: '',
      status: status,
      position: position,
    );

/// One root with a child per remaining status: every state appears once.
final _nodes = [
  _node('Alpha', null, 'done', 0),
  _node('c1', 'Alpha', 'pending', 0),
  _node('c2', 'Alpha', 'in_progress', 1),
  _node('c3', 'Alpha', 'locked', 2),
];

Future<void> _pump(WidgetTester tester, Widget child, String locale) async {
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(home: Scaffold(body: child)),
    ),
  );
  await tester.pumpAndSettle();
}

double _fontSize(WidgetTester tester, Finder text) =>
    tester.renderObject<RenderParagraph>(text).text.style!.fontSize!;

void main() {
  testWidgets('mark labels are announced in the active locale', (tester) async {
    final semantics = tester.ensureSemantics();
    await _pump(
      tester,
      RoadmapMinimap(nodes: _nodes, onTapBand: (_) {}),
      'zh-CN',
    );

    expect(find.bySemanticsLabel('完成'), findsOneWidget);
    expect(find.bySemanticsLabel('待办'), findsOneWidget);
    expect(find.bySemanticsLabel('进行中'), findsOneWidget);
    expect(find.bySemanticsLabel('锁定'), findsOneWidget);
    expect(find.bySemanticsLabel('done'), findsNothing);
    semantics.dispose();
  });

  testWidgets('minimap column titles stay on the 12px floor', (tester) async {
    await _pump(
      tester,
      RoadmapMinimap(nodes: _nodes, onTapBand: (_) {}),
      'en-US',
    );

    final titles = find.descendant(
      of: find.byType(RoadmapMinimap),
      matching: find.text('Alpha'),
    );
    expect(titles, findsOneWidget);
    expect(_fontSize(tester, titles), greaterThanOrEqualTo(12));
  });

  testWidgets('a collaborator cursor label stays on the 12px floor', (
    tester,
  ) async {
    await _pump(
      tester,
      RoadmapCanvas(
        nodes: _nodes,
        controller: ScrollController(),
        tileBuilder: (node, _) => Text(node.title),
        cursors: [CursorDot(sessionId: 's1', name: 'Dana', x: 20, y: 20)],
      ),
      'en-US',
    );

    expect(_fontSize(tester, find.text('Dana')), greaterThanOrEqualTo(12));
  });

  testWidgets('five bands with full-width titles fit a 390px phone', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final roots = [
      for (var i = 0; i < 5; i++) _node('路线规划方案$i', null, 'pending', i),
    ];
    await _pump(
      tester,
      RoadmapMinimap(nodes: roots, onTapBand: (_) {}),
      'zh-CN',
    );

    expect(tester.takeException(), isNull);
  });
}
