import 'package:flutter/material.dart';

import 'cobalt.dart';

/// Rounded card on the field — the Cobalt grammar unit.
class FieldCard extends StatelessWidget {
  const FieldCard({
    super.key,
    required this.child,
    this.onTap,
    this.color,
    this.padding = const EdgeInsets.all(24),
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
        side: BorderSide(color: borderColor ?? palette.strokeSubtle),
      ),
      clipBehavior: Clip.antiAlias,
      child: Padding(padding: padding, child: child),
    );
    if (onTap == null) return card;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(Cobalt.radiusCard),
      child: card,
    );
  }
}
