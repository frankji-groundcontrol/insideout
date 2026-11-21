import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';
import 'roadmap_graph_layout.dart';

/// The tree's connectors, behind the boxes: from a parent's bottom edge down
/// to a rail halfway to the next level, along the rail over its children, and
/// down into each child's top edge. Square corners, 2px, in the strong stroke.
class RoadmapTreePainter extends CustomPainter {
  RoadmapTreePainter(this.layout, this.color);

  final RoadmapGraphLayout layout;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderContainer;
    for (final link in layout.links) {
      final parent = layout.boxes[link.parent]!;
      final rail = parent.bottom + RoadmapGraphLayout.levelGap / 2;
      final kids = [for (final id in link.children) layout.boxes[id]!];
      canvas.drawLine(
        parent.bottomCenter,
        Offset(parent.center.dx, rail),
        paint,
      );
      canvas.drawLine(
        Offset(kids.first.center.dx, rail),
        Offset(kids.last.center.dx, rail),
        paint,
      );
      for (final kid in kids) {
        canvas.drawLine(Offset(kid.center.dx, rail), kid.topCenter, paint);
      }
    }
  }

  @override
  bool shouldRepaint(RoadmapTreePainter old) =>
      old.layout != layout || old.color != color;
}

/// The dependency arrows, over the boxes: a dashed curve in the brand hue from
/// the node waited on to the waiting node, arrowhead at the waiting end. It
/// leaves and enters a quarter box off centre, clear of the tree connectors,
/// and a pair on one level dips through the gap below it.
class RoadmapDependencyPainter extends CustomPainter {
  RoadmapDependencyPainter(this.layout, this.edges, this.color);

  final RoadmapGraphLayout layout;
  final List<({String from, String to})> edges;
  final Color color;

  static const _dash = 6.0;
  static const _gap = 4.0;
  static const _head = 10.0;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderContainer;
    for (final e in edges) {
      final path = _route(layout.boxes[e.from]!, layout.boxes[e.to]!);
      for (final metric in path.computeMetrics()) {
        for (var d = 0.0; d < metric.length; d += _dash + _gap) {
          canvas.drawPath(metric.extractPath(d, d + _dash), paint);
        }
        final tip = metric.getTangentForOffset(metric.length)!;
        final back = -tip.vector * _head;
        for (final turn in [0.45, -0.45]) {
          final wing = Offset(
            back.dx * math.cos(turn) - back.dy * math.sin(turn),
            back.dx * math.sin(turn) + back.dy * math.cos(turn),
          );
          canvas.drawLine(tip.position, tip.position + wing, paint);
        }
      }
    }
  }

  Path _route(Rect from, Rect to) {
    final dx = RoadmapGraphLayout.boxWidth / 4;
    final bend = RoadmapGraphLayout.levelGap * 0.8;
    final Offset start;
    final Offset end;
    final double out;
    final double into;
    if (to.top > from.bottom) {
      start = from.bottomCenter + Offset(dx, 0);
      end = to.topCenter + Offset(dx, 0);
      (out, into) = (bend, -bend);
    } else if (to.bottom < from.top) {
      start = from.topCenter + Offset(dx, 0);
      end = to.bottomCenter + Offset(dx, 0);
      (out, into) = (-bend, bend);
    } else {
      start = from.bottomCenter + Offset(dx, 0);
      end = to.bottomCenter + Offset(dx, 0);
      (out, into) = (bend, bend);
    }
    return Path()
      ..moveTo(start.dx, start.dy)
      ..cubicTo(
        start.dx,
        start.dy + out,
        end.dx,
        end.dy + into,
        end.dx,
        end.dy,
      );
  }

  @override
  bool shouldRepaint(RoadmapDependencyPainter old) =>
      old.layout != layout || old.edges != edges || old.color != color;
}
