import 'package:flutter/material.dart';

/// CSS `cubic-bezier(0.16, 1, 0.3, 1)` from the Nuxt Ink & Seal landing / AuthDoor.
const inkEase = Cubic(0.16, 1.0, 0.3, 1.0);

bool inkReduceMotion(BuildContext context) => MediaQuery.disableAnimationsOf(context);

/// Fade + rise (and optional scale) on mount, or when first scrolled into view.
class InkReveal extends StatefulWidget {
  const InkReveal({
    super.key,
    required this.child,
    this.delay = Duration.zero,
    this.duration = const Duration(milliseconds: 650),
    this.dy = 24,
    this.scaleFrom = 1,
    this.whenVisible = false,
  });

  final Widget child;
  final Duration delay;
  final Duration duration;
  final double dy;
  final double scaleFrom;
  final bool whenVisible;

  @override
  State<InkReveal> createState() => _InkRevealState();
}

class _InkRevealState extends State<InkReveal> with SingleTickerProviderStateMixin {
  late final AnimationController _c;
  ScrollNotificationObserverState? _scroll;
  var _started = false;

  @override
  void initState() {
    super.initState();
    _c = AnimationController(vsync: this, duration: widget.delay + widget.duration);
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
    if (widget.whenVisible && !_inView()) return;
    _started = true;
    _c.forward();
  }

  bool _inView() {
    final box = context.findRenderObject() as RenderBox?;
    if (box == null || !box.hasSize) return false;
    final top = box.localToGlobal(Offset.zero).dy;
    return top < MediaQuery.sizeOf(context).height - 80;
  }

  @override
  void dispose() {
    _scroll?.removeListener(_onScroll);
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final start = widget.delay.inMilliseconds / _c.duration!.inMilliseconds;
    final t = CurvedAnimation(parent: _c, curve: Interval(start, 1, curve: inkEase));
    return AnimatedBuilder(
      animation: t,
      builder: (context, child) {
        final v = t.value;
        return Opacity(
          opacity: v.clamp(0, 1),
          child: Transform.translate(
            offset: Offset(0, widget.dy * (1 - v)),
            child: Transform.scale(
              scale: widget.scaleFrom + (1 - widget.scaleFrom) * v,
              child: child,
            ),
          ),
        );
      },
      child: widget.child,
    );
  }
}

/// Seal stamp: 1.6 → 0.88 → 1.07 → 1, matching AuthDoor / hero seal keyframes.
class InkStamp extends StatefulWidget {
  const InkStamp({
    super.key,
    required this.child,
    this.delay = const Duration(milliseconds: 150),
    this.duration = const Duration(milliseconds: 650),
  });

  final Widget child;
  final Duration delay;
  final Duration duration;

  @override
  State<InkStamp> createState() => _InkStampState();
}

class _InkStampState extends State<InkStamp> with SingleTickerProviderStateMixin {
  late final AnimationController _c;

  @override
  void initState() {
    super.initState();
    _c = AnimationController(vsync: this, duration: widget.delay + widget.duration);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (inkReduceMotion(context)) {
        _c.value = 1;
      } else {
        _c.forward();
      }
    });
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final start = widget.delay.inMilliseconds / _c.duration!.inMilliseconds;
    final t = CurvedAnimation(parent: _c, curve: Interval(start, 1, curve: inkEase));
    return AnimatedBuilder(
      animation: t,
      builder: (context, child) {
        final k = sealStampAt(t.value);
        return Opacity(
          opacity: k.opacity,
          child: Transform.scale(scale: k.scale, child: child),
        );
      },
      child: widget.child,
    );
  }
}

class InkKeyframe {
  const InkKeyframe(this.opacity, this.scale);
  final double opacity;
  final double scale;
}

InkKeyframe assemblePopAt(double t) {
  if (t <= 0) return const InkKeyframe(0, 0.25);
  if (t < 0.6) {
    final u = t / 0.6;
    return InkKeyframe(u, 0.25 + (1.12 - 0.25) * u);
  }
  final u = (t - 0.6) / 0.4;
  return InkKeyframe(1, 1.12 + (1 - 1.12) * u);
}

InkKeyframe sealStampAt(double t) {
  if (t <= 0) return const InkKeyframe(0, 1.6);
  if (t < 0.55) {
    final u = t / 0.55;
    return InkKeyframe(u, 1.6 + (0.88 - 1.6) * u);
  }
  if (t < 0.78) {
    final u = (t - 0.55) / 0.23;
    return InkKeyframe(1, 0.88 + (1.07 - 0.88) * u);
  }
  final u = (t - 0.78) / 0.22;
  return InkKeyframe(1, 1.07 + (1 - 1.07) * u);
}
