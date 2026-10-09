import 'package:flutter/material.dart';

import '../widgets/pressable_block.dart';
import 'block_chrome.dart';
import 'block_focus.dart';
import 'cobalt.dart';
import 'cobalt_theme.dart';

enum BlockButtonVariant { primary, secondary, danger, ghost }

/// Fixed outer heights, border included, measured on the 4px grid (D1).
enum BlockButtonSize {
  sm(EdgeInsets.symmetric(horizontal: 12), 12, 32),
  md(EdgeInsets.symmetric(horizontal: 16), 14, 40),
  lg(EdgeInsets.symmetric(horizontal: 24), 16, 52);

  const BlockButtonSize(this.padding, this.fontSize, this.height);

  final EdgeInsets padding;
  final double fontSize;
  final double height;
}

/// The press-the-block button (D1). Choreography, reproduced exactly:
/// rest `cast` (4,4) — hover `castSm` (2,2) + translate (2,2) — active no
/// shadow + translate (4,4), all over [Cobalt.pressDuration]. Ghost is flat:
/// no border, no shadow, no travel.
class BlockButton extends StatefulWidget {
  const BlockButton({
    super.key,
    required this.label,
    this.onPressed,
    this.variant = BlockButtonVariant.primary,
    this.size = BlockButtonSize.md,
    this.icon,
    this.trailingIcon,
    this.busy = false,
    this.expand = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final BlockButtonVariant variant;
  final BlockButtonSize size;
  final IconData? icon;
  final IconData? trailingIcon;
  final bool busy;
  final bool expand;

  @override
  State<BlockButton> createState() => _BlockButtonState();
}

class _BlockButtonState extends State<BlockButton> {
  var _hover = false;
  var _down = false;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final brightness = Theme.of(context).brightness;
    final enabled = widget.onPressed != null && !widget.busy;

    // Primary and danger are edged in ink, not in the accent, in BOTH themes
    // — that hard edge against the cobalt is the reference's defining detail.
    final (bg, fg, border) = switch (widget.variant) {
      BlockButtonVariant.primary => (
        palette.btn,
        palette.btnFg,
        Cobalt.lightInk,
      ),
      BlockButtonVariant.secondary => (
        palette.surfaceRaised,
        palette.fgPrimary,
        palette.strokeSubtle,
      ),
      BlockButtonVariant.danger => (
        palette.brandHover,
        palette.btnFg,
        Cobalt.lightInk,
      ),
      BlockButtonVariant.ghost => (
        Colors.transparent,
        palette.fgPrimary,
        Colors.transparent,
      ),
    };

    final flat = widget.variant == BlockButtonVariant.ghost;
    final travel = !enabled || flat
        ? 0.0
        : _down
        ? Cobalt.activeOffset.dx
        : _hover
        ? Cobalt.hoverOffset.dx
        : 0.0;
    final shadows = !enabled || flat || _down
        ? const <BoxShadow>[]
        : _hover
        ? castSm(brightness)
        : cast(brightness);

    final labelStyle = TextStyle(
      fontFamily: Cobalt.sans,
      fontFamilyFallback: Cobalt.sansFallback,
      fontSize: widget.size.fontSize,
      fontWeight: FontWeight.w600,
      height: 1.2,
      color: fg,
    );

    Widget box = AnimatedContainer(
      duration: MediaQuery.disableAnimationsOf(context)
          ? Duration.zero
          : Cobalt.pressDuration,
      transform: Matrix4.translationValues(travel, travel, 0),
      height: widget.size.height,
      padding: widget.size.padding,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: bg,
        border: flat
            ? null
            : Border.all(color: border, width: Cobalt.borderContainer),
        boxShadow: shadows,
      ),
      child: Row(
        mainAxisSize: widget.expand ? MainAxisSize.max : MainAxisSize.min,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (widget.busy) ...[
            BlockLoader(size: 14, color: fg),
            const SizedBox(width: Cobalt.space2),
          ] else if (widget.icon != null) ...[
            Icon(widget.icon, size: 16, color: fg),
            const SizedBox(width: Cobalt.space2),
          ],
          if (widget.expand)
            Flexible(
              child: Text(
                widget.label,
                overflow: TextOverflow.ellipsis,
                style: labelStyle,
              ),
            )
          else
            Text(widget.label, style: labelStyle),
          if (widget.trailingIcon != null) ...[
            const SizedBox(width: Cobalt.space2),
            Icon(widget.trailingIcon, size: 16, color: fg),
          ],
        ],
      ),
    );
    if (widget.expand) box = SizedBox(width: double.infinity, child: box);

    final block = Semantics(button: true, enabled: enabled, child: box);
    if (!enabled) {
      return Opacity(
        opacity: 0.5,
        child: MouseRegion(cursor: SystemMouseCursors.basic, child: block),
      );
    }
    return BlockFocus(
      onActivate: widget.onPressed,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hover = true),
        onExit: (_) => setState(() {
          _hover = false;
          _down = false;
        }),
        child: GestureDetector(
          onTap: widget.onPressed,
          onTapDown: (_) => setState(() => _down = true),
          onTapUp: (_) => setState(() => _down = false),
          onTapCancel: () => setState(() => _down = false),
          child: block,
        ),
      ),
    );
  }
}

/// A pressable pick-one-of-a-few block: selected fills with the cobalt under
/// the hard ink edge, unselected stays a raised block. The shared selection
/// grammar (audience switcher, view switchers) — casts and presses via
/// [PressableBlock], never a Material chip ripple.
class BlockChoice extends StatelessWidget {
  const BlockChoice({
    super.key,
    required this.label,
    required this.selected,
    this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return PressableBlock(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(
          horizontal: Cobalt.space3,
          vertical: Cobalt.space2,
        ),
        decoration: BoxDecoration(
          color: selected ? palette.brand : palette.surfaceRaised,
          border: Border.all(
            color: selected ? Cobalt.lightInk : palette.strokeSubtle,
            width: Cobalt.borderContainer,
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontFamily: Cobalt.sans,
            fontFamilyFallback: Cobalt.sansFallback,
            fontSize: 12,
            fontWeight: FontWeight.w600,
            letterSpacing: 0.6,
            color: selected ? palette.btnFg : palette.fgPrimary,
          ),
        ),
      ),
    );
  }
}
