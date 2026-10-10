import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';

// A keyboard-only user scrolls a tall band with the arrow keys and
// PageUp/PageDown while focus sits inside it. Flutter binds the arrows to
// scrolling on the web only (elsewhere they move focus to the next card), so
// the canvas binds Up/Down itself; PageUp/PageDown scroll natively everywhere.

RoadmapNode _node(String id, String? parent, int position) => RoadmapNode(
  id: id,
  projectId: 'p1',
  parentId: parent,
  title: id,
  description: '',
  status: 'pending',
  position: position,
);

/// Band A is taller than the 600px viewport (root + 11 children at 80px).
final _nodes = [
  _node('A', null, 0),
  for (var i = 1; i <= 11; i++) _node('a$i', 'A', i),
  _node('B', null, 1),
];

/// Pumps the canvas and puts focus on a tile inside the tall band.
Future<void> _pumpFocused(WidgetTester tester) async {
  final controller = ScrollController();
  final focus = FocusNode();
  addTearDown(controller.dispose);
  addTearDown(focus.dispose);
  await tester.binding.setSurfaceSize(const Size(400, 600));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: RoadmapCanvas(
          nodes: _nodes,
          controller: controller,
          tileBuilder: (node, _) => Focus(
            focusNode: node.id == 'a1' ? focus : null,
            child: SizedBox(height: 80, child: Text(node.title)),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  focus.requestFocus();
  await tester.pump();
}

/// Scroll offset of the tall band's own vertical list.
double _bandOffset(WidgetTester tester) => tester
    .state<ScrollableState>(
      find
          .descendant(
            of: find.byType(RoadmapCanvas),
            matching: find.byWidgetPredicate(
              (w) => w is Scrollable && w.axis == Axis.vertical,
            ),
          )
          .first,
    )
    .position
    .pixels;

void main() {
  testWidgets('PageDown scrolls the band that holds focus', (tester) async {
    await _pumpFocused(tester);
    expect(_bandOffset(tester), 0);

    await tester.sendKeyEvent(LogicalKeyboardKey.pageDown);
    await tester.pumpAndSettle();

    expect(_bandOffset(tester), greaterThan(0));
  });

  testWidgets('ArrowDown scrolls the band that holds focus, ArrowUp back', (
    tester,
  ) async {
    await _pumpFocused(tester);
    expect(_bandOffset(tester), 0);

    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.pumpAndSettle();
    final scrolled = _bandOffset(tester);
    expect(scrolled, greaterThan(0));

    await tester.sendKeyEvent(LogicalKeyboardKey.arrowUp);
    await tester.pumpAndSettle();
    expect(_bandOffset(tester), lessThan(scrolled));
  });
}
