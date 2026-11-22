import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_windows.dart';

import '_roadmap_canvas_helpers.dart';

// Node windows (2026-11-22): the layout state behind the movable, resizable
// windows a node opens in. The last window in the list is the front one.
void main() {
  const view = Size(1400, 900);

  RoadmapWindows opened(List<String> ids) {
    final w = RoadmapWindows();
    for (final id in ids) {
      w.open(canvasNode(id, null), view);
    }
    return w;
  }

  test('open adds a window in the viewport at the default size', () {
    final w = opened(['a']);
    expect(w.windows.single.node.id, 'a');
    final r = w.windows.single.rect;
    expect(r.size, RoadmapWindows.defaultSize);
    expect((Offset.zero & view).intersect(r), r);
  });

  test('each new window cascades from the last', () {
    final w = opened(['a', 'b']);
    final a = w.windows[0].rect.topLeft;
    final b = w.windows[1].rect.topLeft;
    expect(b - a, const Offset(RoadmapWindows.cascade, RoadmapWindows.cascade));
  });

  test('opening a node that has a window brings it to the front', () {
    final w = opened(['a', 'b']);
    final rect = w.windows[0].rect;
    w.open(canvasNode('a', null), view);
    expect(w.windows.map((x) => x.node.id), ['b', 'a']);
    expect(w.windows.last.rect, rect);
  });

  test('focus brings a window forward; close and closeFront drop one', () {
    final w = opened(['a', 'b', 'c']);
    w.focus('a');
    expect(w.windows.map((x) => x.node.id), ['b', 'c', 'a']);
    w.close('c');
    expect(w.windows.map((x) => x.node.id), ['b', 'a']);
    expect(w.closeFront(), isTrue);
    expect(w.windows.map((x) => x.node.id), ['b']);
    w.close('b');
    expect(w.closeFront(), isFalse);
  });

  test('a drag moves a window but keeps its title bar in view', () {
    final w = opened(['a']);
    final start = w.windows.single.rect;
    w.move('a', const Offset(-40, 30), view);
    expect(
      w.windows.single.rect.topLeft,
      start.topLeft + const Offset(-40, 30),
    );

    w.move('a', const Offset(-5000, -5000), view);
    expect(w.windows.single.rect.topLeft, Offset.zero);

    w.move('a', const Offset(5000, 5000), view);
    final r = w.windows.single.rect;
    expect(r.right, view.width);
    expect(r.top, view.height - RoadmapWindows.titleBar);
  });

  test('a resize holds the minimum and the viewport', () {
    final w = opened(['a']);
    w.move('a', const Offset(-5000, -5000), view);
    w.resize('a', const Offset(-5000, -5000), view);
    expect(w.windows.single.rect.size, RoadmapWindows.minSize);

    w.resize('a', const Offset(5000, 5000), view);
    expect(w.windows.single.rect, Offset.zero & view);
  });

  test('a small viewport opens a window no larger than itself', () {
    final w = RoadmapWindows()
      ..open(canvasNode('a', null), const Size(700, 400));
    final r = w.windows.single.rect;
    expect(r.height, 400);
    expect(r.top, 0);
    expect(r.right, lessThanOrEqualTo(700));
  });
}
