import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_icon_button.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';

/// How far the Canvas zooms, and the margin a fitted flow keeps.
const roadmapMinScale = 0.3;
const roadmapMaxScale = 2.0;
const roadmapViewMargin = 24.0;

/// The whole [graph] in [view]: scaled down to fit (never up past 1x, never
/// under the minimum) and centred.
Matrix4 roadmapFitMatrix(Size graph, Size view) {
  final s = _fitScale(graph, view).clamp(roadmapMinScale, 1.0);
  return Matrix4.translationValues(
    (view.width - graph.width * s) / 2,
    (view.height - graph.height * s) / 2,
    0,
  )..multiply(Matrix4.diagonal3Values(s, s, s));
}

/// Where the Canvas opens: fitted when that still reads (0.75x or more),
/// else at 1x, centred down the view and across it when it is narrower,
/// else with the first column at the left.
Matrix4 roadmapStartMatrix(Size graph, Size view) {
  if (_fitScale(graph, view) >= 0.75) return roadmapFitMatrix(graph, view);
  return Matrix4.translationValues(
    math.max(roadmapViewMargin, (view.width - graph.width) / 2),
    math.max(roadmapViewMargin, (view.height - graph.height) / 2),
    0,
  );
}

double _fitScale(Size graph, Size view) => math.min(
  1.0,
  math.min(
    (view.width - 2 * roadmapViewMargin) / graph.width,
    (view.height - 2 * roadmapViewMargin) / graph.height,
  ),
);

/// [m] zoomed by [factor] about [focal] (a view point that stays put),
/// held between the minimum and maximum scale.
Matrix4 roadmapZoomAbout(Matrix4 m, double factor, Offset focal) {
  final s = m.getMaxScaleOnAxis();
  final f = (s * factor).clamp(roadmapMinScale, roadmapMaxScale) / s;
  return Matrix4.translationValues(focal.dx, focal.dy, 0)
    ..multiply(Matrix4.diagonal3Values(f, f, f))
    ..multiply(Matrix4.translationValues(-focal.dx, -focal.dy, 0))
    ..multiply(m);
}

/// The Canvas's zoom block: in, out and fit, one squared block of house
/// icon buttons on a raised ground.
class RoadmapZoomControls extends StatelessWidget {
  const RoadmapZoomControls({
    super.key,
    required this.onZoomIn,
    required this.onZoomOut,
    required this.onFit,
  });

  final VoidCallback onZoomIn;
  final VoidCallback onZoomOut;
  final VoidCallback onFit;

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final palette = Cobalt.of(context);
    Widget button(String key, IconData icon, String label, VoidCallback on) =>
        BlockIconButton(
          key: Key(key),
          icon: icon,
          tooltip: t(label),
          semanticLabel: t(label),
          onPressed: on,
        );
    return Container(
      padding: const EdgeInsets.all(Cobalt.space1),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(
          color: palette.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
        boxShadow: castSm(Theme.of(context).brightness),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        spacing: Cobalt.space1,
        children: [
          button('roadmap-zoom-in', Icons.add, 'roadmap.zoomIn', onZoomIn),
          button(
            'roadmap-zoom-out',
            Icons.remove,
            'roadmap.zoomOut',
            onZoomOut,
          ),
          button(
            'roadmap-zoom-fit',
            Icons.fit_screen,
            'roadmap.zoomFit',
            onFit,
          ),
        ],
      ),
    );
  }
}
