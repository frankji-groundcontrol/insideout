import 'package:flutter/material.dart';

import 'block_focus.dart';
import 'cobalt.dart';

/// Bordered square icon button on a raised block (D6): 2px `strokeSubtle`
/// border flipping to `brand` on hover, 16px icon, optional 12/w700 label,
/// optional square count badge on the brand.
class BlockIconButton extends StatefulWidget {
  const BlockIconButton({
    super.key,
    required this.icon,
    this.onPressed,
    this.tooltip,
    this.badge,
    this.label,
    this.hoverAccent = true,
    this.semanticLabel,
  });

  final IconData icon;
  final VoidCallback? onPressed;
  final String? tooltip;
  final int? badge;
  final String? label;
  final bool hoverAccent;
  final String? semanticLabel;

  @override
  State<BlockIconButton> createState() => _BlockIconButtonState();
}

class _BlockIconButtonState extends State<BlockIconButton> {
  var _hover = false;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final enabled = widget.onPressed != null;
    final accent = enabled && _hover && widget.hoverAccent;
    final wide = widget.label != null || (widget.badge ?? 0) > 0;
    final fg = accent ? palette.brand : palette.fgSecondary;
    Widget body = BlockFocus(
      enabled: enabled,
      onActivate: widget.onPressed,
      child: MouseRegion(
        cursor: enabled ? SystemMouseCursors.click : SystemMouseCursors.basic,
        onEnter: enabled ? (_) => setState(() => _hover = true) : null,
        onExit: enabled ? (_) => setState(() => _hover = false) : null,
        child: GestureDetector(
          onTap: widget.onPressed,
          child: Semantics(
            button: true,
            enabled: enabled,
            label: widget.semanticLabel,
            child: AnimatedContainer(
              duration: MediaQuery.disableAnimationsOf(context)
                  ? Duration.zero
                  : Cobalt.pressDuration,
              padding: EdgeInsets.symmetric(
                horizontal: wide ? Cobalt.space3 : Cobalt.space2,
                vertical: Cobalt.space2,
              ),
              decoration: BoxDecoration(
                color: palette.surfaceRaised,
                border: Border.all(
                  color: accent ? palette.brand : palette.strokeSubtle,
                  width: Cobalt.borderContainer,
                ),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(widget.icon, size: 16, color: fg),
                  if (widget.label != null) ...[
                    const SizedBox(width: 6),
                    Text(
                      widget.label!,
                      style: TextStyle(
                        fontFamily: Cobalt.sans,
                        fontFamilyFallback: Cobalt.sansFallback,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                        color: fg,
                      ),
                    ),
                  ],
                  if ((widget.badge ?? 0) > 0) ...[
                    const SizedBox(width: Cobalt.space2),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 6,
                        vertical: 2,
                      ),
                      color: palette.brand,
                      child: Text(
                        '${widget.badge}',
                        style: TextStyle(
                          fontFamily: Cobalt.sans,
                          fontFamilyFallback: Cobalt.sansFallback,
                          fontSize: 12,
                          fontWeight: FontWeight.w700,
                          color: palette.btnFg,
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
    if (widget.tooltip == null) return body;
    return Tooltip(message: widget.tooltip!, child: body);
  }
}
