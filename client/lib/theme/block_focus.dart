import 'package:flutter/material.dart';

import 'cobalt.dart';

/// Keyboard focus and Enter/Space activation for the custom blocks, painting
/// the cobalt ring DESIGN.md mandates under Components → Buttons. Space is
/// ActivateIntent on every platform; Enter is ActivateIntent except on the web,
/// where it is ButtonActivateIntent, so both are bound.
///
/// The ring is drawn by a foreground painter outside the child's bounds rather
/// than by an outer border, so wrapping a block never changes its measured
/// size — the fixed control heights the primitive tests assert stay exact.
class BlockFocus extends StatefulWidget {
  const BlockFocus({
    super.key,
    required this.child,
    this.onActivate,
    this.enabled = true,
  });

  final Widget child;
  final VoidCallback? onActivate;

  /// False takes the block out of focus traversal entirely, so a disabled
  /// control is not reachable by Tab.
  final bool enabled;

  @override
  State<BlockFocus> createState() => _BlockFocusState();
}

class _BlockFocusState extends State<BlockFocus> {
  var _focused = false;

  bool get _live => widget.enabled && widget.onActivate != null;

  Object? _activate(Intent _) {
    widget.onActivate?.call();
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return FocusableActionDetector(
      enabled: _live,
      onShowFocusHighlight: (show) => setState(() => _focused = show),
      actions: <Type, Action<Intent>>{
        ActivateIntent: CallbackAction<ActivateIntent>(onInvoke: _activate),
        // On the web the default shortcuts send Enter as this intent, not
        // ActivateIntent ("enter activates buttons, but not other controls"),
        // so a block that listened only for ActivateIntent ignored Enter in a
        // browser while the VM tests, where Enter is ActivateIntent, passed.
        ButtonActivateIntent: CallbackAction<ButtonActivateIntent>(
          onInvoke: _activate,
        ),
      },
      child: CustomPaint(
        foregroundPainter: _focused && _live
            ? CobaltFocusRing(Cobalt.of(context).brand)
            : null,
        child: widget.child,
      ),
    );
  }
}

/// The 2px cobalt ring, offset clear of the block's own ink border, no blur.
class CobaltFocusRing extends CustomPainter {
  const CobaltFocusRing(this.color);

  final Color color;

  /// Gap between the block edge and the ring's inner edge.
  static const gap = 1.0;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawRect(
      (Offset.zero & size).inflate(gap + Cobalt.borderContainer / 2),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = Cobalt.borderContainer
        ..color = color,
    );
  }

  @override
  bool shouldRepaint(CobaltFocusRing oldDelegate) => oldDelegate.color != color;
}
