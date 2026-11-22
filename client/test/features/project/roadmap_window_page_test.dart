import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/features/project/roadmap_window_frame.dart';
import 'package:insideout/features/project/roadmap_windows.dart';

import '_roadmap_canvas_helpers.dart';
import '_roadmap_tree_helpers.dart';

// Node windows (2026-11-22): a tap on a node anywhere on the roadmap page
// opens its information in a movable, resizable window; a double-click, or
// the window's Open roadmap, goes to the node's own roadmap.

/// TreeApi with a glance for every node and Refunds due Now on Progress.
class _WindowApi extends TreeApi {
  _WindowApi() {
    glance = {
      'inCharge': {'userId': 'u-dana', 'name': 'Priya Nair'},
      'now': [
        {'id': 'a1', 'title': 'Refunds', 'status': 'in_progress'},
      ],
    };
  }

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(
        now: [
          ProgressItem.fromJson({
            'id': 'a1',
            'title': 'Refunds',
            'status': 'in_progress',
            'deadline': '2026-12-01',
            'pressure': 'normal',
          }),
        ],
        needsDeadline: [],
        next: [],
        doneCount: 0,
      );
}

Future<GoRouter> _open(WidgetTester tester, {String view = 'canvas'}) async {
  final router = await pumpTreeRouter(tester, treeSession(_WindowApi()));
  router.go('/projects/p1/roadmap?view=$view');
  await tester.pumpAndSettle();
  return router;
}

Finder _window(String id) => find.byKey(Key('roadmap-window-$id'));
String _path(GoRouter r) => r.routerDelegate.currentConfiguration.uri.path;
List<String> _order(WidgetTester tester) => [
  for (final f in tester.widgetList<RoadmapWindowFrame>(
    find.byType(RoadmapWindowFrame),
  ))
    f.node.id,
];

void main() {
  testWidgets('a tap on a step opens its window with the glance', (
    tester,
  ) async {
    final router = await _open(tester);
    await tapNode(tester, stepBox('a'));

    expect(_path(router), '/projects/p1/roadmap');
    expect(_window('a'), findsOneWidget);
    Finder inside(Finder f) => find.descendant(of: _window('a'), matching: f);
    expect(inside(find.text('Payments')), findsOneWidget);
    expect(inside(find.textContaining("PRIYA NAIR'S VIEW")), findsOneWidget);
    expect(inside(find.text('Refunds')), findsOneWidget);
    expect(inside(find.text('Assignment history')), findsOneWidget);
  });

  testWidgets('the title bar drags the window; the corner resizes it', (
    tester,
  ) async {
    await _open(tester);
    await tapNode(tester, stepBox('a'));
    final start = tester.getRect(_window('a'));

    await tester.drag(
      find.byKey(const Key('roadmap-window-bar-a')),
      const Offset(-200, 60),
    );
    await tester.pumpAndSettle();
    final moved = tester.getRect(_window('a'));
    expect(moved.topLeft - start.topLeft, const Offset(-200, 60));

    await tester.drag(
      find.byKey(const Key('roadmap-window-resize-corner')),
      const Offset(-1000, -1000),
    );
    await tester.pumpAndSettle();
    expect(tester.getSize(_window('a')), RoadmapWindows.minSize);
  });

  testWidgets('two windows open; the one touched comes to the front', (
    tester,
  ) async {
    await _open(tester);
    await tapNode(tester, stepBox('a'));
    await tapNode(tester, stepBox('b'));
    expect(_order(tester), ['a', 'b']);

    await tester.tap(find.byKey(const Key('roadmap-window-bar-a')));
    await tester.pumpAndSettle();
    expect(_order(tester), ['b', 'a']);
  });

  testWidgets('the close button and Escape close windows', (tester) async {
    await _open(tester);
    await tapNode(tester, stepBox('a'));
    await tapNode(tester, stepBox('b'));

    await tester.tap(find.byKey(const Key('roadmap-window-close-b')));
    await tester.pumpAndSettle();
    expect(_order(tester), ['a']);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byType(RoadmapWindowFrame), findsNothing);
  });

  testWidgets('a double-click opens the node as its own roadmap', (
    tester,
  ) async {
    final router = await _open(tester);
    await doubleTapNode(tester, stepBox('a'));
    expect(_path(router), '/projects/p1/roadmap/a');
    expect(find.byType(RoadmapWindowFrame), findsNothing);
  });

  testWidgets('Open roadmap goes to the node and closes its windows', (
    tester,
  ) async {
    final router = await _open(tester);
    await tapNode(tester, stepBox('b'));
    await tester.tap(find.byKey(const Key('roadmap-window-open-b')));
    await tester.pumpAndSettle();
    expect(_path(router), '/projects/p1/roadmap/b');
    expect(find.byType(RoadmapWindowFrame), findsNothing);
  });

  testWidgets('a leaf opens a window, not the detail modal', (tester) async {
    await _open(tester);
    await tapNode(tester, stepBox('b'));
    expect(_window('b'), findsOneWidget);
    expect(find.byType(Dialog), findsNothing);
  });

  testWidgets('a Bands card opens its window', (tester) async {
    await _open(tester, view: 'bands');
    await tapNode(
      tester,
      find.descendant(
        of: find.byType(RoadmapNodeCard),
        matching: find.text('Search'),
      ),
    );
    expect(_window('b'), findsOneWidget);
  });

  testWidgets('a Progress item opens its window', (tester) async {
    await _open(tester, view: 'progress');
    await tapNode(tester, find.text('Refunds'));
    expect(_window('a1'), findsOneWidget);
  });

  testWidgets('a phone opens the node as a full-screen sheet', (tester) async {
    await _open(tester, view: 'progress');
    // MediaQuery reads the view, not the surface size.
    tester.view
      ..devicePixelRatio = 1
      ..physicalSize = const Size(400, 800);
    addTearDown(tester.view.reset);
    await tester.binding.setSurfaceSize(const Size(400, 800));
    await tester.pumpAndSettle();
    await tapNode(tester, find.text('Refunds'));

    final sheet = find.byType(Dialog);
    expect(sheet, findsOneWidget);
    expect(tester.getSize(sheet), const Size(400, 800));
    expect(
      find.descendant(of: sheet, matching: find.byType(RoadmapWindowFrame)),
      findsOneWidget,
    );
    expect(
      find.descendant(of: sheet, matching: find.text('Assignment history')),
      findsOneWidget,
    );
    expect(find.byKey(const Key('roadmap-window-resize-corner')), findsNothing);

    await tester.tap(find.byKey(const Key('roadmap-window-close-a1')));
    await tester.pumpAndSettle();
    expect(sheet, findsNothing);
  });
}
