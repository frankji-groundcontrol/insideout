import 'package:flutter/material.dart';

import '../../theme/ink_motion.dart';
import '../../theme/ink_seal.dart';

/// Spark → PRD → tree → seal. Same 400×130 grammar as the Nuxt SVG.
/// Hero mode plays the click-in (pop, then seal stamp) unless reduced-motion.
class AssemblyDiagram extends StatefulWidget {
  const AssemblyDiagram({
    super.key,
    this.mode = AssemblyDiagramMode.progress,
    this.currentStep = 4,
    this.semanticsLabel,
    this.whenVisible = false,
  });

  final AssemblyDiagramMode mode;
  final int currentStep;
  final String? semanticsLabel;
  /// Play the click-in when the widget first enters the viewport (step mini-maps).
  final bool whenVisible;

  @override
  State<AssemblyDiagram> createState() => _AssemblyDiagramState();
}

class _AssemblyDiagramState extends State<AssemblyDiagram> with SingleTickerProviderStateMixin {
  static const _span = Duration(milliseconds: 1500);
  late final AnimationController _c;
  ScrollNotificationObserverState? _scroll;
  var _started = false;

  @override
  void initState() {
    super.initState();
    _c = AnimationController(vsync: this, duration: _span);
    WidgetsBinding.instance.addPostFrameCallback((_) => _tryStart());
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final next = ScrollNotificationObserver.maybeOf(context);
    if (_scroll != next) {
      _scroll?.removeListener(_onScroll);
      _scroll = next;
      _scroll?.addListener(_onScroll);
    }
  }

  void _onScroll(ScrollNotification _) => _tryStart();

  void _tryStart() {
    if (!mounted || _started) return;
    if (inkReduceMotion(context)) {
      _started = true;
      _c.value = 1;
      return;
    }
    if (widget.whenVisible) {
      final box = context.findRenderObject() as RenderBox?;
      if (box == null || !box.hasSize) return;
      final top = box.localToGlobal(Offset.zero).dy;
      if (top >= MediaQuery.sizeOf(context).height - 80) return;
    }
    _started = true;
    _c.forward();
  }

  @override
  void dispose() {
    _scroll?.removeListener(_onScroll);
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = Theme.of(context).brightness == Brightness.dark ? InkSeal.dark : InkSeal.light;
    return Semantics(
      label: widget.semanticsLabel,
      child: AspectRatio(
        aspectRatio: 400 / 130,
        child: AnimatedBuilder(
          animation: _c,
          builder: (context, _) {
            return CustomPaint(
              painter: _AssemblyPainter(
                mode: widget.mode,
                currentStep: widget.currentStep,
                ink: palette.fgPrimary,
                seal: palette.seal,
                muted: palette.strokeSubtle,
                ms: _c.value * _span.inMilliseconds,
                animate: true,
              ),
            );
          },
        ),
      ),
    );
  }
}

enum AssemblyDiagramMode { hero, progress }

class _AssemblyPainter extends CustomPainter {
  _AssemblyPainter({
    required this.mode,
    required this.currentStep,
    required this.ink,
    required this.seal,
    required this.muted,
    required this.ms,
    required this.animate,
  });

  final AssemblyDiagramMode mode;
  final int currentStep;
  final Color ink;
  final Color seal;
  final Color muted;
  final double ms;
  final bool animate;

  static const cx = [45.0, 150.0, 255.0, 345.0];
  static const cy = 65.0;
  static const arrows = [
    (82.0, 113.0),
    (187.0, 218.0),
    (292.0, 322.0),
  ];

  @override
  void paint(Canvas canvas, Size size) {
    canvas.save();
    canvas.scale(size.width / 400);
    for (var i = 0; i < arrows.length; i++) {
      _arrow(canvas, i);
    }
    for (var i = 0; i < 4; i++) {
      canvas.save();
      canvas.translate(cx[i], cy);
      final kf = _nodeKf(i);
      canvas.scale(kf.scale);
      final paint = Paint()
        ..color = _nodeColor(i).withValues(alpha: _nodeOpacity(i) * kf.opacity)
        ..style = PaintingStyle.stroke
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round;
      switch (i) {
        case 0:
          _spark(canvas, paint);
        case 1:
          _doc(canvas, paint);
        case 2:
          _tree(canvas, paint);
        default:
          _seal(canvas, paint);
      }
      canvas.restore();
    }
    canvas.restore();
  }

  Color _nodeColor(int i) {
    if (mode == AssemblyDiagramMode.hero) return i == 3 ? seal : ink;
    if (i == currentStep - 1) return ink;
    if (i < currentStep - 1) return ink;
    return muted;
  }

  double _nodeOpacity(int i) {
    if (mode == AssemblyDiagramMode.hero) return 1;
    if (i < currentStep - 1) return 0.38;
    if (i == currentStep - 1) return 1;
    return 0.55;
  }

  int get _lastLit => mode == AssemblyDiagramMode.hero ? 3 : currentStep - 1;

  InkKeyframe _nodeKf(int i) {
    if (!animate) return const InkKeyframe(1, 1);
    if (i > _lastLit) return const InkKeyframe(1, 1);
    if (i == 3) {
      final t = ((ms - 840) / 650).clamp(0.0, 1.0);
      return sealStampAt(t);
    }
    final t = ((ms - i * 280) / 500).clamp(0.0, 1.0);
    return assemblePopAt(t);
  }

  double _arrowT(int i) {
    if (!animate) return 1;
    if (i + 1 > _lastLit) return 1;
    return ((ms - (i * 2 + 1) * 140) / 350).clamp(0.0, 1.0);
  }

  void _arrow(Canvas canvas, int i) {
    final lit = mode == AssemblyDiagramMode.hero || i + 1 <= currentStep - 1;
    final t = _arrowT(i);
    final paint = Paint()
      ..color = (lit ? ink : muted).withValues(alpha: (lit ? 0.7 : 0.5) * t)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round;
    final x1 = arrows[i].$1;
    final x2 = arrows[i].$1 + (arrows[i].$2 - arrows[i].$1) * t;
    final path = Path()
      ..moveTo(x1, cy)
      ..lineTo(x2 - 6, cy);
    canvas.drawPath(
      path,
      paint..strokeCap = StrokeCap.round,
    );
    // dashed look: overlay with a dash effect by drawing short segments
    final dash = Paint()
      ..color = paint.color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round;
    canvas.drawPath(
      Path()
        ..moveTo(x2 - 6, cy - 4.5)
        ..lineTo(x2, cy)
        ..lineTo(x2 - 6, cy + 4.5),
      dash,
    );
  }

  void _spark(Canvas canvas, Paint stroke) {
    final fill = Paint()..color = stroke.color;
    canvas.drawCircle(Offset.zero, 4.5, fill);
    stroke.strokeWidth = 2.5;
    const rays = [
      (9.0, 0.0, 17.0, 0.0),
      (6.4, -6.4, 12.0, -12.0),
      (0.0, -9.0, 0.0, -17.0),
      (-6.4, -6.4, -12.0, -12.0),
      (-9.0, 0.0, -17.0, 0.0),
      (-6.4, 6.4, -12.0, 12.0),
      (0.0, 9.0, 0.0, 17.0),
      (6.4, 6.4, 12.0, 12.0),
    ];
    for (final r in rays) {
      canvas.drawLine(Offset(r.$1, r.$2), Offset(r.$3, r.$4), stroke);
    }
  }

  void _doc(Canvas canvas, Paint stroke) {
    stroke.strokeWidth = 3;
    canvas.drawPath(
      Path()
        ..moveTo(-15, -21)
        ..lineTo(5, -21)
        ..lineTo(15, -11)
        ..lineTo(15, 21)
        ..lineTo(-15, 21)
        ..close(),
      stroke,
    );
    canvas.drawPath(
      Path()
        ..moveTo(5, -21)
        ..lineTo(5, -11)
        ..lineTo(15, -11),
      stroke,
    );
    stroke.strokeWidth = 2.5;
    canvas.drawLine(const Offset(-9, -3), const Offset(6, -3), stroke);
    canvas.drawLine(const Offset(-9, 4), const Offset(6, 4), stroke);
    canvas.drawLine(const Offset(-9, 11), const Offset(1, 11), stroke);
  }

  void _tree(Canvas canvas, Paint stroke) {
    stroke.strokeWidth = 2.5;
    canvas.drawPath(
      Path()
        ..moveTo(0, -26)
        ..lineTo(-20, -8)
        ..moveTo(0, -26)
        ..lineTo(20, -8)
        ..moveTo(-20, -8)
        ..lineTo(-30, 14)
        ..moveTo(-20, -8)
        ..lineTo(-10, 14)
        ..moveTo(20, -8)
        ..lineTo(10, 14)
        ..moveTo(20, -8)
        ..lineTo(30, 14)
        ..moveTo(-34, 28)
        ..lineTo(34, 28),
      stroke,
    );
    final fill = Paint()..color = stroke.color;
    for (final p in [
      const Offset(0, -26),
      const Offset(-20, -8),
      const Offset(20, -8),
      const Offset(-30, 14),
      const Offset(-10, 14),
      const Offset(10, 14),
      const Offset(30, 14),
    ]) {
      canvas.drawCircle(p, 4, fill);
    }
  }

  void _seal(Canvas canvas, Paint stroke) {
    stroke.strokeWidth = 3.5;
    canvas.drawRRect(RRect.fromLTRBR(-19, -19, 19, 19, const Radius.circular(7)), stroke);
    canvas.drawPath(
      Path()
        ..moveTo(-9, 1)
        ..lineTo(-2, 8)
        ..lineTo(10, -7),
      stroke,
    );
  }

  @override
  bool shouldRepaint(covariant _AssemblyPainter old) =>
      old.mode != mode ||
      old.currentStep != currentStep ||
      old.ink != ink ||
      old.seal != seal ||
      old.ms != ms ||
      old.animate != animate;
}
