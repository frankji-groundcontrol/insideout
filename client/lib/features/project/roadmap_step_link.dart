import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// The handle on a Canvas step's right edge: dragged onto another step, it
/// makes that step wait on this one. Positions are global; the Canvas maps
/// them into its flow.
class RoadmapLinkHandle extends StatelessWidget {
  const RoadmapLinkHandle({
    super.key,
    required this.tooltip,
    required this.onStart,
    required this.onMove,
    required this.onEnd,
  });

  static const size = 16.0;

  final String tooltip;
  final void Function(Offset global) onStart;
  final void Function(Offset global) onMove;
  final VoidCallback onEnd;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Tooltip(
      message: tooltip,
      child: MouseRegion(
        cursor: SystemMouseCursors.grab,
        child: GestureDetector(
          onPanStart: (d) => onStart(d.globalPosition),
          onPanUpdate: (d) => onMove(d.globalPosition),
          onPanEnd: (_) => onEnd(),
          onPanCancel: onEnd,
          child: Container(
            width: size,
            height: size,
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              border: Border.all(
                color: palette.brand,
                width: Cobalt.borderContainer,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// The line a link drag draws, from the step's handle to the pointer.
class RoadmapLinkLinePainter extends CustomPainter {
  RoadmapLinkLinePainter({
    required this.from,
    required this.to,
    required this.color,
  });

  final Offset from;
  final Offset to;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderContainer;
    canvas.drawLine(from, to, paint);
    final back = from - to;
    if (back.distance < 1) return;
    final b = back / back.distance * 10;
    for (final turn in [0.45, -0.45]) {
      final wing = Offset(
        b.dx * math.cos(turn) - b.dy * math.sin(turn),
        b.dx * math.sin(turn) + b.dy * math.cos(turn),
      );
      canvas.drawLine(to, to + wing, paint);
    }
  }

  @override
  bool shouldRepaint(RoadmapLinkLinePainter old) =>
      old.from != from || old.to != to || old.color != color;
}
