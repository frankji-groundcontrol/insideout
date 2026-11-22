import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';

import '_roadmap_canvas_helpers.dart';
import '_roadmap_tree_helpers.dart';

// The Canvas view moves: drag pans, the wheel and the +/- block zoom between
// 0.3x and 2x, Fit shows the whole flow. It opens on a fit when that reads at
// 0.75x or more, else at 1x from the first column. A step opens as its own
// tree, on Progress; a leaf opens its detail.

double _scale(WidgetTester tester) => canvasMatrix(tester).getMaxScaleOnAxis();

Offset _shift(WidgetTester tester) {
  final t = canvasMatrix(tester).getTranslation();
  return Offset(t.x, t.y);
}

Rect _view(WidgetTester tester) =>
    tester.getRect(find.byType(InteractiveViewer));

void main() {
  testWidgets('a flow that fits at 0.75x or more opens fitted and centred', (
    tester,
  ) async {
    await pumpCanvas(tester, canvasChain(3));
    expect(_scale(tester), inInclusiveRange(0.75, 1.0));
    final view = _view(tester);
    final first = stepCard(tester, 'S0');
    final last = stepCard(tester, 'S2');
    expect(first.left - view.left, closeTo(view.right - last.right, 2));
  });

  testWidgets('a long flow opens at 1x from its first column', (tester) async {
    await pumpCanvas(tester, canvasChain(12), size: const Size(800, 600));
    expect(_scale(tester), 1.0);
    final first = stepCard(tester, 'S0');
    expect(first.left - _view(tester).left, inInclusiveRange(8, 48));
  });

  testWidgets('dragging the background pans in both axes', (tester) async {
    await pumpCanvas(tester, canvasChain(12), size: const Size(800, 600));
    final before = _shift(tester);
    await tester.dragFrom(const Offset(400, 40), const Offset(-150, 60));
    await tester.pumpAndSettle();
    final moved = _shift(tester) - before;
    expect(moved.dx, lessThan(-100));
    expect(moved.dy, greaterThan(30));
  });

  testWidgets('the zoom block zooms within 0.3x and 2x, and Fit fits', (
    tester,
  ) async {
    await pumpCanvas(tester, canvasChain(6), size: const Size(800, 600));
    final zoomIn = find.byKey(const Key('roadmap-zoom-in'));
    final zoomOut = find.byKey(const Key('roadmap-zoom-out'));
    await tester.tap(zoomIn);
    await tester.pump();
    expect(_scale(tester), closeTo(1.25, 0.001));
    for (var i = 0; i < 8; i++) {
      await tester.tap(zoomIn);
    }
    await tester.pump();
    expect(_scale(tester), closeTo(2.0, 0.001));
    for (var i = 0; i < 12; i++) {
      await tester.tap(zoomOut);
    }
    await tester.pump();
    expect(_scale(tester), closeTo(0.3, 0.001));

    await tester.tap(find.byKey(const Key('roadmap-zoom-fit')));
    await tester.pump();
    final view = _view(tester);
    final first = stepCard(tester, 'S0');
    final last = stepCard(tester, 'S5');
    expect(first.left, greaterThanOrEqualTo(view.left));
    expect(last.right, lessThanOrEqualTo(view.right));
    expect(_scale(tester), lessThan(1.0));
  });

  testWidgets('the zoom block speaks its buttons', (tester) async {
    final semantics = tester.ensureSemantics();
    await pumpCanvas(tester, canvasChain(2));
    expect(find.bySemanticsLabel('Zoom in'), findsOneWidget);
    expect(find.bySemanticsLabel('Zoom out'), findsOneWidget);
    expect(find.bySemanticsLabel('Fit the whole flow'), findsOneWidget);
    semantics.dispose();
  });

  testWidgets('a double-click opens a step as its own tree, on Progress; a '
      'drag on it pans instead', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap?view=canvas');
    await tester.pumpAndSettle();
    expect(find.byType(RoadmapCanvas), findsOneWidget, reason: 'top route');
    expect(stepBox('a'), findsOneWidget);
    expect(stepBox('r'), findsNothing);

    final before = _shift(tester);
    await tester.drag(find.text('Payments'), const Offset(120, 0));
    await tester.pumpAndSettle();
    expect(
      router.routerDelegate.currentConfiguration.uri.path,
      isNot(contains('/roadmap/a')),
    );
    expect((_shift(tester) - before).dx, greaterThan(60));

    await doubleTapNode(tester, find.text('Payments'));
    final uri = router.routerDelegate.currentConfiguration.uri;
    expect(uri.path, '/projects/p1/roadmap/a');
    expect(uri.queryParameters['view'], isNull);
    expect(find.byType(RoadmapCanvas), findsNothing);
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();
    expect(stepBox('a1'), findsOneWidget);
    expect(stepBox('a'), findsNothing);
  });

  testWidgets('a leaf step opens its window', (tester) async {
    final router = await pumpTreeRouter(tester, treeSession(TreeApi()));
    router.go('/projects/p1/roadmap?view=canvas');
    await tester.pumpAndSettle();
    await tapNode(tester, find.text('Search'));
    expect(find.byKey(const Key('roadmap-window-b')), findsOneWidget);
    expect(
      router.routerDelegate.currentConfiguration.uri.path,
      '/projects/p1/roadmap',
    );
    expect(find.text('Assignment history'), findsOneWidget);
  });
}
