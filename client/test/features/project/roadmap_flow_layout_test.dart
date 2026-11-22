import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/roadmap_flow_layout.dart';

const _step = Size(248, 120);
const _lane = Size(248, 24);

/// Spark: chat waits on matching; launch waits on chat, and across column 1
/// on onboarding (two links), matching and safety through lanes.
final _sparkEdges = [
  (from: 'matching', to: 'chat', count: 1),
  (from: 'chat', to: 'launch', count: 1),
  (from: 'onboarding', to: 'launch', count: 2),
  (from: 'matching', to: 'launch', count: 1),
  (from: 'safety', to: 'launch', count: 1),
];

final _sparkLinks = roadmapFlowLinks(_sparkEdges, {
  _sparkEdges[2]: ['lane1'],
  _sparkEdges[3]: ['lane2'],
  _sparkEdges[4]: ['lane3'],
});

FlowLayout _spark() => roadmapFlowLayout(
  columns: [
    ['onboarding', 'matching', 'safety'],
    ['chat', 'lane1', 'lane2', 'lane3'],
    ['launch'],
  ],
  links: _sparkLinks,
  sizes: {
    for (final id in ['onboarding', 'matching', 'safety', 'chat', 'launch'])
      id: _step,
    for (final id in ['lane1', 'lane2', 'lane3']) id: _lane,
  },
);

Rect _rect(FlowLayout l, String id, Size size) => l.topLeft[id]! & size;

void main() {
  test('Spark lays out uncrossed, unoverlapped and lined up', () {
    final layout = _spark();
    expect(roadmapFlowCrossings(layout.order, _sparkLinks), 0);
    Size sizeOf(String id) => id.startsWith('lane') ? _lane : _step;
    for (final ids in layout.order) {
      final rects = [for (final id in ids) _rect(layout, id, sizeOf(id))];
      for (var i = 1; i < rects.length; i++) {
        expect(rects[i].top - rects[i - 1].bottom, greaterThanOrEqualTo(24));
      }
    }
    double cy(String id) => _rect(layout, id, sizeOf(id)).center.dy;
    expect(cy('chat'), cy('matching'));
    final into = [cy('chat'), cy('lane1'), cy('lane2'), cy('lane3')];
    expect(
      cy('launch'),
      inInclusiveRange(into.reduce(_min), into.reduce(_max)),
    );
    expect(layout.topLeft['launch']!.dx, 2 * (248 + 72));
    expect(layout.size.width, 3 * 248 + 2 * 72);
    final tops = layout.topLeft.values.map((o) => o.dy);
    expect(tops.reduce(_min), 0);
  });

  test('a crossing the given order makes comes out uncrossed', () {
    final links = [('a', 'd', 1), ('b', 'c', 1)];
    final layout = roadmapFlowLayout(
      columns: [
        ['a', 'b'],
        ['c', 'd'],
      ],
      links: links,
      sizes: {for (final id in 'abcd'.split('')) id: _step},
    );
    expect(
      roadmapFlowCrossings([
        ['a', 'b'],
        ['c', 'd'],
      ], links),
      1,
    );
    expect(roadmapFlowCrossings(layout.order, links), 0);
  });

  test('a loop group stays together in its column', () {
    final layout = roadmapFlowLayout(
      columns: [
        ['x', 'y', 'z'],
        ['p', 'q', 'r'],
      ],
      links: [('x', 'p', 1), ('y', 'q', 1), ('z', 'r', 1)],
      sizes: {for (final id in 'xyzpqr'.split('')) id: _step},
      groups: [
        ['p', 'r'],
      ],
    );
    final column = layout.order[1];
    expect((column.indexOf('p') - column.indexOf('r')).abs(), 1);
  });

  test('the same input lays out the same', () {
    final a = _spark();
    final b = _spark();
    expect(a.order, b.order);
    expect(a.topLeft, b.topLeft);
    expect(a.size, b.size);
  });

  test('boxes with no links keep their given order', () {
    final links = [('a', 'd', 1), ('b', 'c', 1)];
    final layout = roadmapFlowLayout(
      columns: [
        ['a', 'b'],
        ['i1', 'c', 'i2', 'd', 'i3'],
      ],
      links: links,
      sizes: {
        for (final id in ['a', 'b', 'c', 'd', 'i1', 'i2', 'i3']) id: _step,
      },
    );
    final column = layout.order[1];
    expect(roadmapFlowCrossings(layout.order, links), 0);
    expect(column.where((id) => id.startsWith('i')), ['i1', 'i2', 'i3']);
    expect(column, ['i1', 'd', 'i2', 'c', 'i3']);
  });
}

double _min(double a, double b) => a < b ? a : b;
double _max(double a, double b) => a > b ? a : b;
