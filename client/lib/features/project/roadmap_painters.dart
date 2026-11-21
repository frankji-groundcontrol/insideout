import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// The flow arrows behind the steps: a 2px curve in the brand hue from the
/// right edge of a step waited on to the left edge of the waiting step, with
/// an arrowhead at the waiting end. An arrow that carries several links says
/// how many in a small squared label near its head. Steps that wait on each
/// other (a loop group, stacked in one column) share one two-headed bracket
/// on their right with a [loopLabel] mark. The boxes' places are read at
/// paint time through [rectOf], after the steps have laid out.
class RoadmapFlowPainter extends CustomPainter {
  RoadmapFlowPainter({
    required this.edges,
    required this.loops,
    required this.loopLabel,
    required this.rectOf,
    required this.color,
    required this.labelStyle,
    required this.labelFill,
  });

  final List<({String from, String to, int count})> edges;
  final List<List<String>> loops;
  final String loopLabel;
  final Rect? Function(String id) rectOf;
  final Color color;
  final TextStyle labelStyle;
  final Color labelFill;

  static const _headLength = 10.0;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderContainer;
    for (final e in edges) {
      final from = rectOf(e.from);
      final to = rectOf(e.to);
      if (from == null || to == null) continue;
      final start = from.centerRight;
      final end = to.centerLeft - const Offset(Cobalt.borderContainer, 0);
      final pull = math.max(40.0, (end.dx - start.dx).abs() / 2);
      final path = Path()
        ..moveTo(start.dx, start.dy)
        ..cubicTo(
          start.dx + pull,
          start.dy,
          end.dx - pull,
          end.dy,
          end.dx,
          end.dy,
        );
      canvas.drawPath(path, paint);
      final metric = path.computeMetrics().first;
      final tip = metric.getTangentForOffset(metric.length)!;
      _head(canvas, tip.position, -tip.vector, paint);
      if (e.count > 1) {
        final at = metric.getTangentForOffset(math.max(0, metric.length - 28))!;
        _label(canvas, '${e.count}', at.position, paint);
      }
    }
    for (final group in loops) {
      final rects = [for (final id in group) rectOf(id)].nonNulls.toList();
      if (rects.length > 1) _loop(canvas, rects, paint);
    }
  }

  /// An arrowhead at [tip], its wings swept back along [back].
  void _head(Canvas canvas, Offset tip, Offset back, Paint paint) {
    final b = back * _headLength;
    for (final turn in [0.45, -0.45]) {
      final wing = Offset(
        b.dx * math.cos(turn) - b.dy * math.sin(turn),
        b.dx * math.sin(turn) + b.dy * math.cos(turn),
      );
      canvas.drawLine(tip, tip + wing, paint);
    }
  }

  /// A loop group's bracket: a spine right of the stacked steps, a tick into
  /// each with a head at both ends, and the loop mark on the spine.
  void _loop(Canvas canvas, List<Rect> rects, Paint paint) {
    final x = rects.map((r) => r.right).reduce(math.max) + Cobalt.space4;
    final top = rects.first.center.dy;
    final bottom = rects.last.center.dy;
    canvas.drawLine(Offset(x, top), Offset(x, bottom), paint);
    for (final (i, r) in rects.indexed) {
      final tip = r.centerRight + const Offset(Cobalt.borderContainer, 0);
      canvas.drawLine(Offset(x, r.center.dy), tip, paint);
      if (i == 0 || i == rects.length - 1) {
        _head(canvas, tip, const Offset(1, 0), paint);
      }
    }
    _label(canvas, loopLabel, Offset(x, (top + bottom) / 2), paint);
  }

  void _label(Canvas canvas, String text, Offset centre, Paint stroke) {
    final tp = TextPainter(
      text: TextSpan(text: text, style: labelStyle),
      textDirection: TextDirection.ltr,
    )..layout();
    final box = Rect.fromCenter(
      center: centre,
      width: tp.width + Cobalt.space2,
      height: tp.height + Cobalt.space1,
    );
    canvas
      ..drawRect(box, Paint()..color = labelFill)
      ..drawRect(box, stroke);
    tp.paint(canvas, box.topLeft + const Offset(Cobalt.space1, 2));
  }

  // The boxes move with layout, not with this painter's fields.
  @override
  bool shouldRepaint(RoadmapFlowPainter old) => true;
}
