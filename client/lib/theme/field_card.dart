import 'package:flutter/material.dart';

import '../widgets/pressable_block.dart';
import 'cobalt.dart';
import 'cobalt_theme.dart';

/// Squared card on the field — the Cobalt grammar unit.
class FieldCard extends StatelessWidget {
  const FieldCard({
    super.key,
    required this.child,
    this.onTap,
    this.color,
    this.padding = const EdgeInsets.all(Cobalt.space6),
    this.borderColor,
  });

  final Widget child;
  final VoidCallback? onTap;
  final Color? color;
  final EdgeInsetsGeometry padding;
  final Color? borderColor;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final card = Material(
      color: color ?? palette.surfaceRaised,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusCard),
        side: BorderSide(
          color: borderColor ?? palette.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
      ),
      clipBehavior: Clip.antiAlias,
      child: Padding(padding: padding, child: child),
    );
    if (onTap == null) {
      return Container(
        decoration: BoxDecoration(
          boxShadow: cast(Theme.of(context).brightness),
        ),
        child: card,
      );
    }
    // D7: press-the-block replaces the InkWell ripple, and PressableBlock
    // owns the cast — the card drops its own so no double shadow lands.
    return PressableBlock(onTap: onTap, child: card);
  }
}
