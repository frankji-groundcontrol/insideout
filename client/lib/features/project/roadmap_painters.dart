import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// Dashed square outline for locked minimap marks. Stroke-only paint;
/// the dash is drawn as short segments because BoxDecoration has no
/// dashed border.
class MinimapDashMarkPainter extends CustomPainter {
  MinimapDashMarkPainter(this.color);

  final Color color;

  static const _dash = 2.0;
  static const _gap = 1.0;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = Cobalt.borderDivider;
    // Inset by half the stroke so the outline is not clipped at the edges.
    final rect = Rect.fromLTWH(
      Cobalt.borderDivider / 2,
      Cobalt.borderDivider / 2,
      size.width - Cobalt.borderDivider,
      size.height - Cobalt.borderDivider,
    );
    var x = rect.left;
    while (x < rect.right) {
      final end = (x + _dash).clamp(rect.left, rect.right);
      canvas.drawLine(Offset(x, rect.top), Offset(end, rect.top), paint);
      canvas.drawLine(Offset(x, rect.bottom), Offset(end, rect.bottom), paint);
      x += _dash + _gap;
    }
    var y = rect.top;
    while (y < rect.bottom) {
      final end = (y + _dash).clamp(rect.top, rect.bottom);
      canvas.drawLine(Offset(rect.left, y), Offset(rect.left, end), paint);
      canvas.drawLine(Offset(rect.right, y), Offset(rect.right, end), paint);
      y += _dash + _gap;
    }
  }

  @override
  bool shouldRepaint(MinimapDashMarkPainter oldDelegate) =>
      oldDelegate.color != color;
}
