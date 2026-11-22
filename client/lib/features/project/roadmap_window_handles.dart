import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';
import 'roadmap_windows.dart';

/// A node window's resize handles, laid over its card: the right edge, the
/// bottom edge and the bottom-right corner, each with its resize cursor.
/// [onResize] takes the growth from the bottom-right.
List<Widget> roadmapWindowHandles(void Function(Offset delta) onResize) {
  const edge = 8.0;
  const corner = 16.0;
  Widget handle(
    String key,
    MouseCursor cursor,
    Offset Function(Offset d) axis, {
    Widget? child,
  }) => MouseRegion(
    cursor: cursor,
    child: GestureDetector(
      key: Key(key),
      behavior: HitTestBehavior.opaque,
      onPanUpdate: (d) => onResize(axis(d.delta)),
      child: child,
    ),
  );
  return [
    Positioned(
      right: 0,
      top: RoadmapWindows.titleBar,
      bottom: corner,
      width: edge,
      child: handle(
        'roadmap-window-resize-right',
        SystemMouseCursors.resizeLeftRight,
        (d) => Offset(d.dx, 0),
      ),
    ),
    Positioned(
      left: 0,
      right: corner,
      bottom: 0,
      height: edge,
      child: handle(
        'roadmap-window-resize-bottom',
        SystemMouseCursors.resizeUpDown,
        (d) => Offset(0, d.dy),
      ),
    ),
    Positioned(
      right: 0,
      bottom: 0,
      width: corner,
      height: corner,
      child: handle(
        'roadmap-window-resize-corner',
        SystemMouseCursors.resizeDownRight,
        (d) => d,
        child: const _Grip(),
      ),
    ),
  ];
}

/// The corner's grip: two squared ticks in the quiet ink.
class _Grip extends StatelessWidget {
  const _Grip();

  @override
  Widget build(BuildContext context) {
    final color = Cobalt.of(context).fgSecondary;
    Widget tick(double w, double h) =>
        Container(width: w, height: h, color: color);
    return Align(
      alignment: Alignment.bottomRight,
      child: Padding(
        padding: const EdgeInsets.all(3),
        child: Stack(
          alignment: Alignment.bottomRight,
          children: [
            tick(8, Cobalt.borderContainer),
            tick(Cobalt.borderContainer, 8),
          ],
        ),
      ),
    );
  }
}
