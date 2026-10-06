import 'package:flutter/material.dart';

import '../theme/cobalt.dart';
import '../theme/cobalt_theme.dart';

/// Press-the-block (Cobalt-Neo decree): rest sits on the full cast shadow,
/// hover shifts [Cobalt.hoverOffset] with the cast shrunk to `castSm`, active
/// presses the block into the page — [Cobalt.activeOffset], shadow gone —
/// all over [Cobalt.pressDuration]. Shadows come from `cast`/`castSm`.
class PressableBlock extends StatefulWidget {
  const PressableBlock({super.key, required this.child, this.onTap});

  final Widget child;
  final VoidCallback? onTap;

  @override
  State<PressableBlock> createState() => _PressableBlockState();
}

class _PressableBlockState extends State<PressableBlock> {
  var _hovered = false;
  var _pressed = false;

  bool get _enabled => widget.onTap != null;

  void _set({bool? hovered, bool? pressed}) {
    setState(() {
      if (hovered != null) _hovered = hovered;
      if (pressed != null) _pressed = pressed;
    });
  }

  @override
  Widget build(BuildContext context) {
    final enabled = _enabled;
    final pressed = enabled && _pressed;
    final hovered = enabled && _hovered && !pressed;
    final brightness = Theme.of(context).brightness;
    final offset = pressed
        ? Cobalt.activeOffset
        : hovered
        ? Cobalt.hoverOffset
        : Offset.zero;
    final shadows = pressed
        ? const <BoxShadow>[]
        : hovered
        ? castSm(brightness)
        : cast(brightness);
    return MouseRegion(
      onEnter: enabled ? (_) => _set(hovered: true) : null,
      onExit: enabled ? (_) => _set(hovered: false, pressed: false) : null,
      child: GestureDetector(
        onTap: widget.onTap,
        onTapDown: enabled ? (_) => _set(pressed: true) : null,
        onTapUp: enabled ? (_) => _set(pressed: false) : null,
        onTapCancel: enabled ? () => _set(pressed: false) : null,
        child: Semantics(
          button: true,
          enabled: enabled,
          child: AnimatedContainer(
            duration: MediaQuery.disableAnimationsOf(context)
                ? Duration.zero
                : Cobalt.pressDuration,
            transform: Matrix4.translationValues(offset.dx, offset.dy, 0),
            decoration: BoxDecoration(boxShadow: shadows),
            child: widget.child,
          ),
        ),
      ),
    );
  }
}
